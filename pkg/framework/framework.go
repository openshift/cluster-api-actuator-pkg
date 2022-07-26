package framework

import (
	"context"
	"errors"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	cov1helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Various constants used by E2E tests.
const (
	PollNodesReadyTimeout   = 10 * time.Minute
	ClusterKey              = "machine.openshift.io/cluster-api-cluster"
	MachineSetKey           = "machine.openshift.io/cluster-api-machineset"
	MachineAPINamespace     = "openshift-machine-api"
	GlobalInfrastuctureName = "cluster"
	WorkerNodeRoleLabel     = "node-role.kubernetes.io/worker"
	RetryShort              = 1 * time.Second
	RetryMedium             = 5 * time.Second
	// DefaultMachineSetReplicas is the default number of replicas of a machineset
	// if MachineSet.Spec.Replicas field is set to nil
	DefaultMachineSetReplicas = 0
	MachinePhaseRunning       = "Running"
	MachinePhaseFailed        = "Failed"
	MachineRoleLabel          = "machine.openshift.io/cluster-api-machine-role"
	MachineTypeLabel          = "machine.openshift.io/cluster-api-machine-type"
	MachineAnnotationKey      = "machine.openshift.io/machine"
)

var (
	WaitShort    = 1 * time.Minute
	WaitMedium   = 3 * time.Minute
	WaitLong     = 15 * time.Minute
	WaitOverLong = 30 * time.Minute
)

// GetInfrastructure fetches the global cluster infrastructure object.
func GetInfrastructure(c runtimeclient.Client) (*configv1.Infrastructure, error) {
	infra := &configv1.Infrastructure{}
	infraName := runtimeclient.ObjectKey{
		Name: GlobalInfrastuctureName,
	}

	if err := c.Get(context.Background(), infraName, infra); err != nil {
		return nil, err
	}

	return infra, nil
}

var platform configv1.PlatformType

// GetPlatform fetches the PlatformType from the infrastructure object.
// Caches value after first successful retrieval.
func GetPlatform(c runtimeclient.Client) (configv1.PlatformType, error) {
	// platform won't change during test run and might be cached
	if platform != "" {
		return platform, nil
	}
	infra, err := GetInfrastructure(c)
	if err != nil {
		return "", err
	}
	if infra.Status.PlatformStatus == nil {
		return "", errors.New("platform status is not populated in infrastructure object")
	}
	platform = infra.Status.PlatformStatus.Type
	return platform, nil
}

// LoadClient returns a new controller-runtime client.
func LoadClient() (runtimeclient.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return runtimeclient.New(cfg, runtimeclient.Options{})
}

// LoadClientset returns a new Kubernetes Clientset.
func LoadClientset() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func WaitForStatusAvailableShort(client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(client, name, WaitShort)
}

func WaitForStatusAvailableMedium(client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(client, name, WaitMedium)
}

func WaitForStatusAvailableOverLong(client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(client, name, WaitOverLong)
}

func expectStatusAvailableIn(client runtimeclient.Client, name string, timeout time.Duration) bool {
	key := types.NamespacedName{
		Name: name,
	}
	clusterOperator := &configv1.ClusterOperator{}

	if err := wait.PollImmediate(RetryMedium, timeout, func() (bool, error) {
		if err := client.Get(context.TODO(), key, clusterOperator); err != nil {
			klog.Errorf("error querying api for OperatorStatus object: %v, retrying...", err)
			return false, nil
		}
		if cov1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorAvailable) {
			klog.Errorf("Condition: %q is false", configv1.OperatorAvailable)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorProgressing) {
			klog.Errorf("Condition: %q is true", configv1.OperatorProgressing)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorDegraded) {
			klog.Errorf("Condition: %q is true", configv1.OperatorDegraded)
			return false, nil
		}
		return true, nil
	}); err != nil {
		klog.Errorf("Error checking isStatusAvailable: %v", err)
		return false
	}
	return true
}

func WaitForValidatingWebhook(client runtimeclient.Client, name string) bool {
	key := types.NamespacedName{Name: name}
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	if err := wait.PollImmediate(RetryShort, WaitShort, func() (bool, error) {
		if err := client.Get(context.TODO(), key, webhook); err != nil {
			klog.Errorf("error querying api for ValidatingWebhookConfiguration: %v, retrying...", err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		klog.Errorf("Error waiting for ValidatingWebhookConfiguration: %v", err)
		return false
	}

	return true
}

// WaitForEvent expects to find the given event
func WaitForEvent(c runtimeclient.Client, kind, name, reason string) error {
	return wait.PollImmediate(RetryMedium, WaitMedium, func() (bool, error) {
		eventList := corev1.EventList{}
		if err := c.List(context.Background(), &eventList); err != nil {
			klog.Errorf("error querying api for eventList object: %v, retrying...", err)
			return false, nil
		}

		for _, event := range eventList.Items {
			if event.Reason != reason ||
				event.InvolvedObject.Kind != kind ||
				event.InvolvedObject.Name != name {
				continue
			}

			return true, nil
		}

		return false, nil
	})
}
