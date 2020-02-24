package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	cov1helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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
	WaitShort               = 1 * time.Minute
	WaitMedium              = 3 * time.Minute
	WaitLong                = 15 * time.Minute
	RetryMedium             = 5 * time.Second
	// DefaultMachineSetReplicas is the default number of replicas of a machineset
	// if MachineSet.Spec.Replicas field is set to nil
	DefaultMachineSetReplicas = 0
	MachinePhaseRunning       = "Running"
	MachineRoleLabel          = "machine.openshift.io/cluster-api-machine-role"
	MachineTypeLabel          = "machine.openshift.io/cluster-api-machine-type"
	MachineAnnotationKey      = "machine.openshift.io/machine"
)

// DeleteObjectsByLabels list all objects of a given kind by labels and deletes them.
// Currently supported kinds:
// - caov1beta1.MachineAutoscalerList
// - caov1.ClusterAutoscalerList
// - batchv1.JobList
func DeleteObjectsByLabels(c runtimeclient.Client, labels map[string]string, list runtime.Object) error {
	if err := c.List(context.Background(), list, runtimeclient.MatchingLabels(labels)); err != nil {
		return fmt.Errorf("Unable to list objects: %v", err)
	}

	// TODO(jchaloup): find a way how to list the items independent of a kind
	var objs []runtime.Object
	switch d := list.(type) {
	case *caov1beta1.MachineAutoscalerList:
		for _, item := range d.Items {
			objs = append(objs, runtime.Object(&item))
		}
	case *caov1.ClusterAutoscalerList:
		for _, item := range d.Items {
			objs = append(objs, runtime.Object(&item))
		}
	case *batchv1.JobList:
		for _, item := range d.Items {
			objs = append(objs, runtime.Object(&item))
		}

	default:
		return fmt.Errorf("List type %#v not recognized", list)
	}

	cascadeDelete := metav1.DeletePropagationForeground
	for _, obj := range objs {
		if err := c.Delete(context.Background(), obj, &runtimeclient.DeleteOptions{
			PropagationPolicy: &cascadeDelete,
		}); err != nil {
			return fmt.Errorf("error deleting object: %v", err)
		}
	}

	return nil
}

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

// RandomString returns a random 6 character string.
func RandomString(clusterName string) string {
	randID := string(uuid.NewUUID())

	return fmt.Sprintf("%s-%s", clusterName, randID[:6])
}

func IsStatusAvailable(client runtimeclient.Client, name string) bool {
	key := types.NamespacedName{
		Namespace: MachineAPINamespace,
		Name:      name,
	}
	clusterOperator := &configv1.ClusterOperator{}

	if err := wait.PollImmediate(1*time.Second, WaitShort, func() (bool, error) {
		if err := client.Get(context.TODO(), key, clusterOperator); err != nil {
			glog.Errorf("error querying api for OperatorStatus object: %v, retrying...", err)
			return false, nil
		}
		if cov1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorAvailable) {
			glog.Errorf("Condition: %q is false", configv1.OperatorAvailable)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorProgressing) {
			glog.Errorf("Condition: %q is true", configv1.OperatorProgressing)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorDegraded) {
			glog.Errorf("Condition: %q is true", configv1.OperatorDegraded)
			return false, nil
		}
		return true, nil
	}); err != nil {
		glog.Errorf("Error checking isStatusAvailable: %v", err)
		return false
	}
	return true
}

func WaitForValidatingWebhook(client runtimeclient.Client, name string) bool {
	key := types.NamespacedName{Name: name}
	webhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}

	if err := wait.PollImmediate(1*time.Second, WaitShort, func() (bool, error) {
		if err := client.Get(context.TODO(), key, webhook); err != nil {
			glog.Errorf("error querying api for ValidatingWebhookConfiguration: %v, retrying...", err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		glog.Errorf("Error waiting for ValidatingWebhookConfiguration: %v", err)
		return false
	}

	return true
}
