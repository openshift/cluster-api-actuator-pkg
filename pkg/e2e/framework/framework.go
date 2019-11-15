package framework

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Default timeout for pools
	PoolTimeout = 5 * time.Minute
	// Default waiting interval for pools
	PollInterval = 5 * time.Second
	// Node waiting internal
	PollNodeInterval = 5 * time.Second
	// Pool timeout for cluster API deployment
	PoolClusterAPIDeploymentTimeout = 10 * time.Minute
	PoolDeletionTimeout             = 1 * time.Minute
	// Pool timeout for kubeconfig
	PoolKubeConfigTimeout = 10 * time.Minute
	PoolNodesReadyTimeout = 10 * time.Minute
	// Instances are running timeout
	TimeoutPoolMachineRunningInterval = 10 * time.Minute

	ClusterKey    = "machine.openshift.io/cluster-api-cluster"
	MachineSetKey = "machine.openshift.io/cluster-api-machineset"
)

// ClusterID set by -cluster-id flag
var ClusterID string

// Path to private ssh to connect to instances (e.g. to download kubeconfig or copy docker images)
var sshkey string

// Default ssh user
var sshuser string

var machineControllerImage string
var machineManagerImage string
var nodelinkControllerImage string

var libvirtURI string
var libvirtPK string

type TestContextType struct {
	KubeConfig          string
	MachineApiNamespace string
	Host                string
}

var TestContext TestContextType

func init() {
	flag.StringVar(&TestContext.KubeConfig, "kubeconfig", "", "kubeconfig file")
	flag.StringVar(&TestContext.MachineApiNamespace, "machine-api-namespace", "openshift-machine-api", "Default machine API namespace")
	flag.StringVar(&ClusterID, "cluster-id", "", "cluster ID")
	flag.StringVar(&sshkey, "ssh-key", "", "Path to private ssh to connect to instances (e.g. to download kubeconfig or copy docker images)")
	flag.StringVar(&sshuser, "ssh-user", "ec2-user", "Ssh user to connect to instances")
	flag.StringVar(&machineControllerImage, "machine-controller-image", "gcr.io/k8s-cluster-api/machine-controller:0.0.1", "Machine controller (actuator) image to run")
	flag.StringVar(&machineManagerImage, "machine-manager-image", "gcr.io/k8s-cluster-api/machine-controller:0.0.1", "Machine manager image to run")
	flag.StringVar(&nodelinkControllerImage, "nodelink-controller-image", "gcr.io/k8s-cluster-api/machine-controller:0.0.1", "Nodelink controller image to run")

	// libvirt specific flags
	flag.StringVar(&libvirtURI, "libvirt-uri", "", "Libvirt URI to connect to libvirt from within machine controller container")
	flag.StringVar(&libvirtPK, "libvirt-pk", "", "Private key to connect to qemu+ssh libvirt uri")

	flag.Parse()
}

// RestclientConfig builds a REST client config
func RestclientConfig() (*clientcmdapi.Config, error) {
	glog.Infof(">>> kubeConfig: %s", TestContext.KubeConfig)
	if TestContext.KubeConfig == "" {
		return nil, fmt.Errorf("KubeConfig must be specified to load client config")
	}
	c, err := clientcmd.LoadFromFile(TestContext.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading KubeConfig: %v", err.Error())
	}
	return c, nil
}

// LoadConfig builds config from kubernetes config
func LoadConfig() (*rest.Config, error) {
	c, err := RestclientConfig()
	if err != nil {
		if TestContext.KubeConfig == "" {
			return rest.InClusterConfig()
		}
		return nil, err
	}
	return clientcmd.NewDefaultClientConfig(*c, &clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: TestContext.Host}}).ClientConfig()
}

// LoadClient builds controller runtime client that accepts any registered type
func LoadClient() (runtimeclient.Client, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err.Error())
	}
	return runtimeclient.New(config, runtimeclient.Options{})
}

func LoadRestClient() (*rest.RESTClient, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err.Error())
	}
	configShallowCopy := *config
	gv := corev1.SchemeGroupVersion
	configShallowCopy.GroupVersion = &gv
	configShallowCopy.APIPath = "/api"
	configShallowCopy.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	rc, err := rest.RESTClientFor(&configShallowCopy)
	if err != nil {
		return nil, fmt.Errorf("unable to build rest client: %v", err)
	}
	return rc, nil
}

func LoadClientset() (*kubernetes.Clientset, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err.Error())
	}
	return kubernetes.NewForConfig(config)
}

func IsNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func WaitUntilAllNodesAreReady(client runtimeclient.Client) error {
	return wait.PollImmediate(1*time.Second, PoolNodesReadyTimeout, func() (bool, error) {
		nodeList := corev1.NodeList{}
		if err := client.List(context.TODO(), &nodeList); err != nil {
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
