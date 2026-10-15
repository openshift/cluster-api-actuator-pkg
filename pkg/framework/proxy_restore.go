package framework

import (
	"context"
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProxyClusterState records the proxy configuration, pools and nodes that
// existed before the proxy test.
type ProxyClusterState struct {
	proxy *configv1.Proxy
	pools map[string]string
	nodes []string
}

// poolRollback records the proxy-enabled target immediately before restoration.
// A pool which has already retained its original target needs no new rollout.
type poolRollback struct {
	rendered   string
	generation int64
	changed    bool
}

func machineConfigObject(kind, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "machineconfiguration.openshift.io", Version: "v1", Kind: kind})
	obj.SetName(name)

	return obj
}

// CaptureProxyClusterState must be called before modifying the cluster proxy.
// It does not wait for a proxy-enabled rollout or impose a setup health gate.
func CaptureProxyClusterState(ctx context.Context, c client.Client) (*ProxyClusterState, error) {
	state := &ProxyClusterState{proxy: &configv1.Proxy{}, pools: map[string]string{}}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, state.proxy); err != nil {
		return nil, err
	}

	pools := &unstructured.UnstructuredList{}
	pools.SetGroupVersionKind(schema.GroupVersionKind{Group: "machineconfiguration.openshift.io", Version: "v1", Kind: "MachineConfigPoolList"})

	if err := c.List(ctx, pools); err != nil {
		return nil, err
	}

	for _, pool := range pools.Items {
		name, _, err := unstructured.NestedString(pool.Object, "spec", "configuration", "name")
		if err != nil {
			return nil, fmt.Errorf("reading pool %s configuration: %w", pool.GetName(), err)
		}

		if name == "" {
			return nil, fmt.Errorf("pool %s has no rendered configuration", pool.GetName())
		}

		state.pools[pool.GetName()] = name
	}

	nodes := &corev1.NodeList{}
	if err := c.List(ctx, nodes); err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		state.nodes = append(state.nodes, node.Name)
	}

	return state, nil
}

// RestoreProxyAndWaitForClusterReady restores the original Proxy spec and waits for propagation, completed
// pool rollouts and usable pre-existing nodes. MITM must remain available until
// this returns successfully. All retries share one bounded cleanup wait.
func (s *ProxyClusterState) RestoreProxyAndWaitForClusterReady(ctx context.Context, c client.Client) error {
	By("Restoring the saved cluster-wide Proxy spec")

	var (
		lastErr  error
		rollback map[string]poolRollback
	)

	By("Waiting for proxy restoration to propagate, MachineConfigPools to fully settle, and pre-existing nodes to be Ready and schedulable")

	// Proxy rollback can trigger control-plane and worker rollouts that take
	// longer than WaitOverLong; wait for nodes to return to service before
	// gather-extra runs its e2e-analysis readiness checks.
	err := wait.PollUntilContextTimeout(ctx, RetryMedium, 2*WaitOverLong, true, func(ctx context.Context) (bool, error) {
		lastErr = nil

		// Capture the current targets before writing the restored Proxy spec.
		// This is a cleanup-only observation, not a setup rollout gate.
		if rollback == nil {
			rollback, lastErr = s.rollbackTargets(ctx, c)
		}

		if lastErr == nil {
			lastErr = s.restoreProxy(ctx, c, rollback)
		}

		if lastErr != nil {
			klog.Infof("Waiting for proxy rollback: %v", lastErr)
		}

		return lastErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("proxy rollback did not finish: %w (last observation: %w)", err, lastErr)
	}

	return nil
}

// restoreProxy performs one attempt. The caller retries both API errors and
// observations of incomplete propagation within the shared cleanup deadline.
func (s *ProxyClusterState) restoreProxy(ctx context.Context, c client.Client, rollback map[string]poolRollback) error {
	proxy := &configv1.Proxy{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, proxy); err != nil {
		return err
	}

	if !reflect.DeepEqual(proxy.Spec, s.proxy.Spec) {
		proxy.Spec = *s.proxy.Spec.DeepCopy()
		if err := c.Update(ctx, proxy); err != nil {
			return err
		}

		return fmt.Errorf("original proxy spec restored; waiting for propagation")
	}

	return s.restored(ctx, c, proxy, rollback)
}

func (s *ProxyClusterState) rollbackTargets(ctx context.Context, c client.Client) (map[string]poolRollback, error) {
	targets := map[string]poolRollback{}

	for name, original := range s.pools {
		pool := machineConfigObject("MachineConfigPool", name)
		if err := c.Get(ctx, client.ObjectKeyFromObject(pool), pool); err != nil {
			return nil, err
		}

		rendered, _, err := unstructured.NestedString(pool.Object, "spec", "configuration", "name")
		if err != nil {
			return nil, err
		}

		targets[name] = poolRollback{rendered: rendered, generation: pool.GetGeneration(), changed: rendered != original}
	}

	return targets, nil
}

func (s *ProxyClusterState) restored(ctx context.Context, c client.Client, proxy *configv1.Proxy, rollback map[string]poolRollback) error {
	if !reflect.DeepEqual(proxy.Status, s.proxy.Status) {
		return fmt.Errorf("proxy status has not returned to its original values")
	}

	controller := machineConfigObject("ControllerConfig", "machine-config-controller")
	if err := c.Get(ctx, client.ObjectKeyFromObject(controller), controller); err != nil {
		return err
	}
	// Check the live proxy input and rendering progress, not equality with a
	// saved ControllerConfig or trust bundle that may have changed independently.
	if err := proxyRenderingComplete(controller, proxy.Status); err != nil {
		return err
	}

	for name, target := range rollback {
		pool := machineConfigObject("MachineConfigPool", name)
		if err := c.Get(ctx, client.ObjectKeyFromObject(pool), pool); err != nil {
			return err
		}

		if err := poolRestored(pool, target); err != nil {
			return err
		}
	}

	for _, name := range s.nodes {
		node := &corev1.Node{}
		if err := c.Get(ctx, client.ObjectKey{Name: name}, node); err != nil {
			return err
		}

		if !nodeRestored(node) {
			return fmt.Errorf("pre-existing node %s is not Ready, schedulable and network available", name)
		}
	}

	return nil
}

func proxyRenderingComplete(controller *unstructured.Unstructured, status configv1.ProxyStatus) error {
	settings, _, err := unstructured.NestedFieldNoCopy(controller.Object, "spec", "proxy")
	if err != nil {
		return err
	}

	if settings == nil {
		settings = map[string]interface{}{}
	}

	input, ok := settings.(map[string]interface{})
	if !ok {
		return fmt.Errorf("template controller proxy input is not an object")
	}

	for field, expected := range map[string]string{"httpProxy": status.HTTPProxy, "httpsProxy": status.HTTPSProxy, "noProxy": status.NoProxy} {
		value, _, err := unstructured.NestedString(input, field)
		if err != nil || value != expected {
			return fmt.Errorf("template controller has not observed restored proxy field %s", field)
		}
	}

	if !generationObserved(controller) || !hasConditions(controller, map[string]string{
		"TemplateControllerCompleted": "True", "TemplateControllerRunning": "False", "TemplateControllerFailing": "False",
	}) {
		return fmt.Errorf("template controller has not completed rendering the restored proxy settings")
	}

	return nil
}

func generationObserved(obj *unstructured.Unstructured) bool {
	observed, found, err := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	return err == nil && found && observed == obj.GetGeneration()
}

func hasConditions(obj *unstructured.Unstructured, expected map[string]string) bool {
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return false
	}

	for conditionType, status := range expected {
		found := false

		for _, entry := range conditions {
			condition, ok := entry.(map[string]interface{})
			if ok && condition["type"] == conditionType && condition["status"] == status {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func poolRestored(pool *unstructured.Unstructured, rollback poolRollback) error {
	desired, _, _ := unstructured.NestedString(pool.Object, "spec", "configuration", "name")
	applied, _, _ := unstructured.NestedString(pool.Object, "status", "configuration", "name")

	if rollback.changed && (desired == rollback.rendered || pool.GetGeneration() <= rollback.generation) {
		return fmt.Errorf("pool %s still reports its pre-rollback configuration %s", pool.GetName(), rollback.rendered)
	}

	if desired == "" || applied != desired || !generationObserved(pool) ||
		!hasConditions(pool, map[string]string{"Updated": "True", "Updating": "False", "Degraded": "False"}) {
		return fmt.Errorf("pool %s has not observed and applied its desired configuration (desired %s, applied %s, status %v)", pool.GetName(), desired, applied, pool.Object["status"])
	}

	count, found, err := unstructured.NestedInt64(pool.Object, "status", "machineCount")
	if err != nil || !found {
		return fmt.Errorf("pool %s has no machine count", pool.GetName())
	}

	for _, field := range []string{"updatedMachineCount", "readyMachineCount", "unavailableMachineCount", "degradedMachineCount"} {
		value, found, err := unstructured.NestedInt64(pool.Object, "status", field)

		want := int64(0)
		if field == "updatedMachineCount" || field == "readyMachineCount" {
			want = count
		}

		if err != nil || !found || value != want {
			return fmt.Errorf("pool %s is not settled: %s=%d, want %d", pool.GetName(), field, value, want)
		}
	}

	return nil
}

// Match Origin's e2e node predicate: Ready, not cordoned, and NetworkUnavailable
// either absent or False. Other taints and pressure conditions are not blockers
// in that predicate.
func nodeRestored(node *corev1.Node) bool {
	if !IsNodeReady(node) || node.Spec.Unschedulable {
		return false
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeNetworkUnavailable && condition.Status != corev1.ConditionFalse {
			return false
		}
	}

	return true
}
