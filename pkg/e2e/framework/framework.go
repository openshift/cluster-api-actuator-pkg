package framework

import (
	"context"
	"time"

	"github.com/golang/glog"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Various constants used by E2E tests.
const (
	PollNodesReadyTimeout = 10 * time.Minute
	ClusterKey            = "machine.openshift.io/cluster-api-cluster"
	MachineSetKey         = "machine.openshift.io/cluster-api-machineset"
	MachineAPINamespace   = "openshift-machine-api"
)

// LoadClient returns a new controller-runtime client.
func LoadClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{})
}

// LoadRestClient returns a new RESTClient.
func LoadRestClient() (*rest.RESTClient, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	rc, err := rest.UnversionedRESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// LoadClientset returns a new Kubernetes Clientset.
func LoadClientset() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

// IsNodeReady returns true if the given node is ready.
func IsNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// WaitUntilAllNodesAreReady lists all nodes and waits until they are ready.
func WaitUntilAllNodesAreReady(c client.Client) error {
	return wait.PollImmediate(1*time.Second, PollNodesReadyTimeout, func() (bool, error) {
		nodeList := corev1.NodeList{}
		if err := c.List(context.TODO(), &nodeList); err != nil {
			glog.Errorf("error querying api for nodeList object: %v, retrying...", err)
			return false, nil
		}
		// All nodes needs to be ready
		for _, node := range nodeList.Items {
			if !IsNodeReady(&node) {
				glog.Errorf("Node %q is not ready", node.Name)
				return false, nil
			}
		}
		return true, nil
	})
}

// NewMachineSet returns a new MachineSet object.
func NewMachineSet(
	clusterName, namespace, name string,
	selectorLabels map[string]string,
	templateLabels map[string]string,
	providerSpec *mapiv1beta1.ProviderSpec,
	replicas int32,
) *mapiv1beta1.MachineSet {
	ms := mapiv1beta1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ClusterKey: clusterName,
			},
		},
		Spec: mapiv1beta1.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					ClusterKey:    clusterName,
					MachineSetKey: name,
				},
			},
			Template: mapiv1beta1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						ClusterKey:    clusterName,
						MachineSetKey: name,
					},
				},
				Spec: mapiv1beta1.MachineSpec{
					ProviderSpec: *providerSpec.DeepCopy(),
				},
			},
			Replicas: pointer.Int32Ptr(replicas),
		},
	}

	// Copy additional labels but do not overwrite those that
	// already exist.
	for k, v := range selectorLabels {
		if _, exists := ms.Spec.Selector.MatchLabels[k]; !exists {
			ms.Spec.Selector.MatchLabels[k] = v
		}
	}
	for k, v := range templateLabels {
		if _, exists := ms.Spec.Template.ObjectMeta.Labels[k]; !exists {
			ms.Spec.Template.ObjectMeta.Labels[k] = v
		}
	}

	return &ms
}
