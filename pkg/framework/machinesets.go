package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// MachineSetParams represents the parameters for creating a new MachineSet
// resource for use in tests.
type MachineSetParams struct {
	Name         string
	Replicas     int32
	Labels       map[string]string
	ProviderSpec *mapiv1beta1.ProviderSpec
}

const (
	machineAPIGroup = "machine.openshift.io"
)

// CreateMachineSet creates a new MachineSet resource.
func CreateMachineSet(c client.Client, params MachineSetParams) (*mapiv1beta1.MachineSet, error) {
	if params.Labels == nil {
		params.Labels = make(map[string]string)
	}

	// TODO(bison): It would be nice to automatically set the Cluster ID / name
	// in the labels here, but I'm not sure how to easily find it.
	params.Labels["e2e.openshift.io"] =
		fmt.Sprintf("%s-%d", params.Name, time.Now().Unix())

	ms := &mapiv1beta1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: MachineAPINamespace,
			Labels:    params.Labels,
		},
		Spec: mapiv1beta1.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: params.Labels,
			},
			Template: mapiv1beta1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: params.Labels,
				},
				Spec: mapiv1beta1.MachineSpec{
					ProviderSpec: *params.ProviderSpec,
				},
			},
			Replicas: pointer.Int32Ptr(params.Replicas),
		},
	}

	if err := c.Create(context.Background(), ms); err != nil {
		return nil, err
	}

	return ms, nil
}

// GetMachineSets gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachineSets(client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]mapiv1beta1.MachineSet, error) {
	machineSetList := &mapiv1beta1.MachineSetList{}

	listOpts := append([]runtimeclient.ListOption{},
		runtimeclient.InNamespace(MachineAPINamespace),
	)

	for _, selector := range selectors {
		s, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}

		listOpts = append(listOpts,
			runtimeclient.MatchingLabelsSelector{Selector: s},
		)
	}

	if err := client.List(context.Background(), machineSetList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineSetList object: %v", err)
	}

	return machineSetList.Items, nil
}

// GetMachineSet gets a machineset by its name from the default machine API namespace.
func GetMachineSet(client runtimeclient.Client, name string) (*mapiv1beta1.MachineSet, error) {
	machineSet := &mapiv1beta1.MachineSet{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := client.Get(context.Background(), key, machineSet); err != nil {
		return nil, fmt.Errorf("error querying api for machineSet object: %v", err)
	}

	return machineSet, nil
}

// GetWorkerMachineSets returns the MachineSets that label their Machines with
// the "worker" role.
func GetWorkerMachineSets(client runtimeclient.Client) ([]*mapiv1beta1.MachineSet, error) {
	machineSets := &mapiv1beta1.MachineSetList{}

	if err := client.List(context.Background(), machineSets); err != nil {
		return nil, err
	}

	var result []*mapiv1beta1.MachineSet

	// The OpenShift installer does not label MachinSets with a type or role,
	// but the Machines themselves are labled as such via the template., so we
	// can reach into the template and check the lables there.
	for i, ms := range machineSets.Items {
		labels := ms.Spec.Template.GetLabels()

		if labels[MachineRoleLabel] == "worker" {
			result = append(result, &machineSets.Items[i])
		}
	}

	if len(result) < 1 {
		return nil, fmt.Errorf("no worker MachineSets found")
	}

	return result, nil
}

// GetMachinesFromMachineSet returns an array of machines owned by a given machineSet
func GetMachinesFromMachineSet(client runtimeclient.Client, machineSet *mapiv1beta1.MachineSet) ([]*mapiv1beta1.Machine, error) {
	machines, err := GetMachines(client)
	if err != nil {
		return nil, fmt.Errorf("error getting machines: %v", err)
	}
	var machinesForSet []*mapiv1beta1.Machine
	for key := range machines {
		if metav1.IsControlledBy(machines[key], machineSet) {
			machinesForSet = append(machinesForSet, machines[key])
		}
	}
	return machinesForSet, nil
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

// ScaleMachineSet scales a machineSet with a given name to the given number of replicas
func ScaleMachineSet(name string, replicas int) error {
	scaleClient, err := getScaleClient()
	if err != nil {
		return fmt.Errorf("error calling getScaleClient %v", err)
	}

	scale, err := scaleClient.Scales(MachineAPINamespace).Get(schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, name)
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales get: %v", err)
	}

	scaleUpdate := scale.DeepCopy()
	scaleUpdate.Spec.Replicas = int32(replicas)
	_, err = scaleClient.Scales(MachineAPINamespace).Update(schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, scaleUpdate)
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales update: %v", err)
	}
	return nil
}

// getScaleClient returns a ScalesGetter object to manipulate scale subresources
func getScaleClient() (scale.ScalesGetter, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting config %v", err)
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("error calling NewDiscoveryRESTMapper %v", err)
	}

	discovery := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discovery)
	scaleClient, err := scale.NewForConfig(cfg, mapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	if err != nil {
		return nil, fmt.Errorf("error calling building scale client %v", err)
	}
	return scaleClient, nil
}

// WaitForMachineSet waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready.
func WaitForMachineSet(c client.Client, name string) {
	machineSet, err := GetMachineSet(c, name)
	Expect(err).ToNot(HaveOccurred())

	selector := machineSet.Spec.Selector

	Eventually(func() error {
		machines, err := GetMachines(c, &selector)
		if err != nil {
			return err
		}

		replicas := pointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0)

		if len(machines) != int(replicas) {
			return fmt.Errorf("found %d Machines, but MachineSet has %d replicas",
				len(machines), machineSet.Spec.Replicas)
		}

		running := FilterRunningMachines(machines)

		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			return fmt.Errorf("not all Machines are running")
		}

		for _, m := range running {
			node, err := GetNodeForMachine(c, m)
			if err != nil {
				return err
			}

			if !IsNodeReady(node) {
				return fmt.Errorf("%s: node is not ready", node.Name)
			}
		}

		return nil
	}, WaitLong, RetryMedium).ShouldNot(HaveOccurred())
}

// WaitForMachineSetDelete polls until the given MachineSet is not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetDelete(c runtimeclient.Client, machineSet *mapiv1beta1.MachineSet) {
	Eventually(func() bool {
		selector := machineSet.Spec.Selector

		machines, err := GetMachines(c, &selector)
		if err != nil || len(machines) != 0 {
			return false // Still have Machines, or other error.
		}

		err = c.Get(context.Background(), runtimeclient.ObjectKey{
			Name:      machineSet.GetName(),
			Namespace: machineSet.GetNamespace(),
		}, &mapiv1beta1.MachineSet{})

		if !apierrors.IsNotFound(err) {
			return false // MachineSet not deleted, or other error.
		}

		return true // MachineSet and Machines were deleted.
	}, WaitLong, RetryMedium).Should(BeTrue())
}
