package framework

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const restoredRendered = "rendered-worker-restored-with-other-changes"

func poolStatus(pool *unstructured.Unstructured) map[string]interface{} {
	status, ok := pool.Object["status"].(map[string]interface{})
	if !ok {
		panic("test fixture has no pool status")
	}

	return status
}

func restoredPool() *unstructured.Unstructured {
	pool := machineConfigObject("MachineConfigPool", "worker")
	pool.SetGeneration(3)
	pool.Object["spec"] = map[string]interface{}{"configuration": map[string]interface{}{"name": restoredRendered}}
	pool.Object["status"] = map[string]interface{}{
		"observedGeneration": int64(3),
		"configuration":      map[string]interface{}{"name": restoredRendered},
		"machineCount":       int64(2), "updatedMachineCount": int64(2), "readyMachineCount": int64(2),
		"unavailableMachineCount": int64(0), "degradedMachineCount": int64(0),
		"conditions": []interface{}{
			map[string]interface{}{"type": "Updated", "status": "True"},
			map[string]interface{}{"type": "Updating", "status": "False"},
			map[string]interface{}{"type": "Degraded", "status": "False"},
		},
	}

	return pool
}

func TestPoolRestored(t *testing.T) {
	rollback := poolRollback{rendered: "rendered-worker-mitm", generation: 2, changed: true}

	tests := []struct {
		name   string
		mutate func(*unstructured.Unstructured)
	}{
		{"healthy earlier configuration", func(p *unstructured.Unstructured) {
			p.Object["spec"] = map[string]interface{}{"configuration": map[string]interface{}{"name": rollback.rendered}}
			poolStatus(p)["configuration"] = map[string]interface{}{"name": rollback.rendered}
		}},
		{"rollback in flight", func(p *unstructured.Unstructured) {
			poolStatus(p)["configuration"] = map[string]interface{}{"name": rollback.rendered}
		}},
		{"unchanged generation", func(p *unstructured.Unstructured) {
			p.SetGeneration(2)
			poolStatus(p)["observedGeneration"] = int64(2)
		}},
		{"stale observed generation", func(p *unstructured.Unstructured) {
			poolStatus(p)["observedGeneration"] = int64(2)
		}},
	}
	for _, field := range []string{"updatedMachineCount", "readyMachineCount", "unavailableMachineCount", "degradedMachineCount"} {
		tests = append(tests, struct {
			name   string
			mutate func(*unstructured.Unstructured)
		}{field, func(p *unstructured.Unstructured) { poolStatus(p)[field] = int64(1) }})
	}

	for _, condition := range []string{"Updated", "Updating", "Degraded"} {
		tests = append(tests, struct {
			name   string
			mutate func(*unstructured.Unstructured)
		}{condition, func(p *unstructured.Unstructured) {
			status := "True"
			if condition == "Updated" {
				status = "False"
			}

			conditions := []interface{}{
				map[string]interface{}{"type": "Updated", "status": "True"},
				map[string]interface{}{"type": "Updating", "status": "False"},
				map[string]interface{}{"type": "Degraded", "status": "False"},
			}

			for i, name := range []string{"Updated", "Updating", "Degraded"} {
				if name == condition {
					conditions[i] = map[string]interface{}{"type": condition, "status": status}
				}
			}

			poolStatus(p)["conditions"] = conditions
		}})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := restoredPool()
			test.mutate(pool)

			if err := poolRestored(pool, rollback); err == nil {
				t.Fatal("accepted an unsettled or stale pool")
			}
		})
	}

	if err := poolRestored(restoredPool(), rollback); err != nil {
		t.Fatalf("rejected restored pool with a new rendered name: %v", err)
	}

	// Failed setup may never have changed a pool. Do not require a new rollout
	// once rendering has observed the restored proxy input.
	if err := poolRestored(restoredPool(), poolRollback{rendered: restoredRendered, generation: 3}); err != nil {
		t.Fatalf("rejected unchanged pool: %v", err)
	}
}

func TestProxyRollbackNodeReadyAndSchedulable(t *testing.T) {
	for _, test := range []struct {
		name     string
		ready    corev1.ConditionStatus
		cordoned bool
		network  corev1.ConditionStatus
		want     bool
	}{
		{name: "Ready with no network condition", ready: corev1.ConditionTrue, want: true},
		{name: "Ready but cordoned", ready: corev1.ConditionTrue, cordoned: true},
		{name: "not Ready", ready: corev1.ConditionFalse},
		{name: "Ready unknown", ready: corev1.ConditionUnknown},
		{name: "network available", ready: corev1.ConditionTrue, network: corev1.ConditionFalse, want: true},
		{name: "network unavailable", ready: corev1.ConditionTrue, network: corev1.ConditionTrue},
		{name: "network unknown", ready: corev1.ConditionTrue, network: corev1.ConditionUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			node := &corev1.Node{Spec: corev1.NodeSpec{Unschedulable: test.cordoned}, Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: test.ready}},
			}}
			if test.network != "" {
				node.Status.Conditions = append(node.Status.Conditions, corev1.NodeCondition{Type: corev1.NodeNetworkUnavailable, Status: test.network})
			}

			if got := nodeRestored(node); got != test.want {
				t.Fatalf("nodeRestored() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProxyRenderingComplete(t *testing.T) {
	controller := machineConfigObject("ControllerConfig", "machine-config-controller")
	controller.SetGeneration(4)
	controller.Object["spec"] = map[string]interface{}{
		"proxy": nil, "additionalTrustBundle": "an independently changed trust bundle",
	}

	controller.Object["status"] = map[string]interface{}{
		"observedGeneration": int64(4),
		"conditions": []interface{}{
			map[string]interface{}{"type": "TemplateControllerCompleted", "status": "True"},
			map[string]interface{}{"type": "TemplateControllerRunning", "status": "False"},
			map[string]interface{}{"type": "TemplateControllerFailing", "status": "False"},
		},
	}
	if err := proxyRenderingComplete(controller, configv1.ProxyStatus{}); err != nil {
		t.Fatalf("rejected an observed empty proxy with unrelated trust changes: %v", err)
	}

	if err := proxyRenderingComplete(controller, configv1.ProxyStatus{HTTPProxy: "http://original:8080"}); err == nil {
		t.Fatal("accepted rendering that has not observed the restored proxy")
	}

	controller.Object["spec"] = map[string]interface{}{"proxy": map[string]interface{}{"httpProxy": "http://original:8080"}}
	if err := proxyRenderingComplete(controller, configv1.ProxyStatus{HTTPProxy: "http://original:8080"}); err != nil {
		t.Fatalf("rejected the restored non-empty proxy: %v", err)
	}

	controller.SetGeneration(5)

	if err := proxyRenderingComplete(controller, configv1.ProxyStatus{HTTPProxy: "http://original:8080"}); err == nil {
		t.Fatal("accepted stale template-controller status")
	}
}
