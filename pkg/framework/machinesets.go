package framework

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	vsphere "github.com/openshift/machine-api-operator/pkg/apis/vsphereprovider/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/klog"
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

func BuildMachineSetParams(client runtimeclient.Client, replicas int) MachineSetParams {
	// Get the current workers MachineSets so we can copy a ProviderSpec
	// from one to use with our new dedicated MachineSet.
	workers, err := GetWorkerMachineSets(client)
	Expect(err).ToNot(HaveOccurred())

	providerSpec := workers[0].Spec.Template.Spec.ProviderSpec.DeepCopy()
	clusterName := workers[0].Spec.Template.Labels[ClusterKey]

	clusterInfra, err := GetInfrastructure(client)
	Expect(err).NotTo(HaveOccurred())
	Expect(clusterInfra.Status.InfrastructureName).ShouldNot(BeEmpty())

	providerSpec, err = prepareProviderSpec(clusterInfra.Status.PlatformStatus.Type, providerSpec)
	Expect(err).NotTo(HaveOccurred())

	uid, err := uuid.NewUUID()
	Expect(err).NotTo(HaveOccurred())

	return MachineSetParams{
		Name:         clusterInfra.Status.InfrastructureName,
		Replicas:     int32(replicas),
		ProviderSpec: providerSpec,
		Labels: map[string]string{
			"e2e.openshift.io": uid.String(),
			ClusterKey:         clusterName,
		},
	}
}
func prepareProviderSpec(platform configv1.PlatformType, ps *mapiv1beta1.ProviderSpec) (*mapiv1beta1.ProviderSpec, error) {
	switch platform {
	case configv1.VSpherePlatformType:
		return prepareVsphereProviderSpec(ps)
	default:
		return ps, nil
	}
}

func prepareVsphereProviderSpec(spec *mapiv1beta1.ProviderSpec) (*mapiv1beta1.ProviderSpec, error) {
	// Shrink vsphere worker machines ram amount during machineset copying.
	// Without this we are spawning 16G ram workers.
	providerSpec := &vsphere.VSphereMachineProviderSpec{}
	err := json.Unmarshal(spec.Value.Raw, providerSpec)
	if err != nil {
		return nil, err
	}
	providerSpec.MemoryMiB = 2048
	return &mapiv1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: providerSpec,
		},
	}, nil
}

// CreateMachineSet creates a new MachineSet resource.
func CreateMachineSet(c client.Client, params MachineSetParams) (*mapiv1beta1.MachineSet, error) {
	ms := &mapiv1beta1.MachineSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineSet",
			APIVersion: "machine.openshift.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: params.Name,
			Namespace:    MachineAPINamespace,
			Labels:       params.Labels,
		},
		Spec: mapiv1beta1.MachineSetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: params.Labels,
			},
			Template: mapiv1beta1.MachineTemplateSpec{
				ObjectMeta: mapiv1beta1.ObjectMeta{
					Labels: params.Labels,
				},
				Spec: mapiv1beta1.MachineSpec{
					ObjectMeta: mapiv1beta1.ObjectMeta{
						Labels: params.Labels,
					},
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
func GetMachineSets(client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]*mapiv1beta1.MachineSet, error) {
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
		return nil, fmt.Errorf("error querying api for machineSetList object: %w", err)
	}

	machineSets := []*mapiv1beta1.MachineSet{}
	for _, ms := range machineSetList.Items {
		machineSet := ms
		machineSets = append(machineSets, &machineSet)
	}

	return machineSets, nil
}

// GetMachineSet gets a machineset by its name from the default machine API namespace.
func GetMachineSet(client runtimeclient.Client, name string) (*mapiv1beta1.MachineSet, error) {
	machineSet := &mapiv1beta1.MachineSet{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := client.Get(context.Background(), key, machineSet); err != nil {
		return nil, fmt.Errorf("error querying api for machineSet object: %w", err)
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
		labels := ms.Spec.Template.ObjectMeta.Labels

		if labels == nil {
			continue
		}

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
		return nil, fmt.Errorf("error getting machines: %w", err)
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
				ObjectMeta: mapiv1beta1.ObjectMeta{
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
		return fmt.Errorf("error calling getScaleClient %w", err)
	}

	scale, err := scaleClient.Scales(MachineAPINamespace).Get(context.Background(), schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales get: %w", err)
	}

	scaleUpdate := scale.DeepCopy()
	scaleUpdate.Spec.Replicas = int32(replicas)
	_, err = scaleClient.Scales(MachineAPINamespace).Update(context.Background(), schema.GroupResource{Group: machineAPIGroup, Resource: "MachineSet"}, scaleUpdate, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error calling scaleClient.Scales update: %w", err)
	}
	return nil
}

// getScaleClient returns a ScalesGetter object to manipulate scale subresources
func getScaleClient() (scale.ScalesGetter, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting config %w", err)
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("error calling NewDiscoveryRESTMapper %w", err)
	}

	discovery := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(discovery)
	scaleClient, err := scale.NewForConfig(cfg, mapper, dynamic.LegacyAPIPathResolverFunc, scaleKindResolver)
	if err != nil {
		return nil, fmt.Errorf("error calling building scale client %w", err)
	}
	return scaleClient, nil
}

// WaitForMachineSet waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready.
func WaitForMachineSet(c client.Client, name string) {
	machineSet, err := GetMachineSet(c, name)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		machines, err := GetMachinesFromMachineSet(c, machineSet)
		if err != nil {
			return err
		}

		replicas := pointer.Int32PtrDerefOr(machineSet.Spec.Replicas, 0)

		if len(machines) != int(replicas) {
			return fmt.Errorf("%q: found %d Machines, but MachineSet has %d replicas",
				name, len(machines), int(replicas))
		}

		running := FilterRunningMachines(machines)

		// This could probably be smarter, but seems fine for now.
		if len(running) != len(machines) {
			return fmt.Errorf("%q: not all Machines are running: %d of %d",
				name, len(running), len(machines))
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
	}, WaitOverLong, RetryMedium).ShouldNot(HaveOccurred())
}

// WaitForMachineSetDelete polls until the given MachineSet is not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetDelete(c runtimeclient.Client, machineSet *mapiv1beta1.MachineSet) {
	WaitForMachineSetsDeleted(c, machineSet)
}

// WaitForMachineSetsDeleted polls until the given MachineSets are not found, and
// there are zero Machines found matching the MachineSet's label selector.
func WaitForMachineSetsDeleted(c runtimeclient.Client, machineSets ...*mapiv1beta1.MachineSet) {
	for _, ms := range machineSets {
		Eventually(func() bool {
			selector := ms.Spec.Selector

			machines, err := GetMachines(c, &selector)
			if err != nil || len(machines) != 0 {
				return false // Still have Machines, or other error.
			}

			err = c.Get(context.Background(), runtimeclient.ObjectKey{
				Name:      ms.GetName(),
				Namespace: ms.GetNamespace(),
			}, &mapiv1beta1.MachineSet{})

			if !apierrors.IsNotFound(err) {
				return false // MachineSet not deleted, or other error.
			}

			return true // MachineSet and Machines were deleted.
		}, WaitLong, RetryMedium).Should(BeTrue())
	}
}

// DeleteMachineSets deletes the specified machinesets and returns an error on failure.
func DeleteMachineSets(client runtimeclient.Client, machineSets ...*mapiv1beta1.MachineSet) error {
	for _, ms := range machineSets {
		if err := client.Delete(context.TODO(), ms); err != nil {
			klog.Errorf("Error querying api for machine object %q: %v, retrying...", ms.Name, err)
			return err
		}
	}
	return nil
}
