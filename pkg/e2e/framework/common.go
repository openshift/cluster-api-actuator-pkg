package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WorkerNodeRoleLabel = "node-role.kubernetes.io/worker"
	WaitShort           = 1 * time.Minute
	WaitMedium          = 3 * time.Minute
	WaitLong            = 15 * time.Minute
	RetryMedium         = 5 * time.Second
	// DefaultMachineSetReplicas is the default number of replicas of a machineset
	// if MachineSet.Spec.Replicas field is set to nil
	DefaultMachineSetReplicas = 0

	MachinePhaseRunning = "Running"
	MachineRoleLabel    = "machine.openshift.io/cluster-api-machine-role"
	MachineTypeLabel    = "machine.openshift.io/cluster-api-machine-type"
)

// GetNodes gets a list of nodes from a running cluster
// Optionaly, labels may be used to constrain listed nodes.
func GetNodes(client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]corev1.Node, error) {
	var listOpts []runtimeclient.ListOption

	nodeList := corev1.NodeList{}

	for _, selector := range selectors {
		s, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}

		listOpts = append(listOpts,
			runtimeclient.MatchingLabelsSelector{Selector: s},
		)
	}

	if err := client.List(context.TODO(), &nodeList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for nodeList object: %v", err)
	}

	return nodeList.Items, nil
}

// GetMachineSets gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachineSets(ctx context.Context, client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]mapiv1beta1.MachineSet, error) {
	machineSetList := &mapiv1beta1.MachineSetList{}

	listOpts := append([]runtimeclient.ListOption{},
		runtimeclient.InNamespace(TestContext.MachineApiNamespace),
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

	if err := client.List(ctx, machineSetList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineSetList object: %v", err)
	}

	return machineSetList.Items, nil
}

// GetMachineSet gets a machineset by its name from the default machine API namespace.
func GetMachineSet(ctx context.Context, client runtimeclient.Client, machineSetName string) (*mapiv1beta1.MachineSet, error) {
	machineSet := &mapiv1beta1.MachineSet{}
	if err := client.Get(ctx, runtimeclient.ObjectKey{Namespace: TestContext.MachineApiNamespace, Name: machineSetName}, machineSet); err != nil {
		return nil, fmt.Errorf("error querying api for machineSet object: %v", err)
	}
	return machineSet, nil
}

// GetMachines gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachines(ctx context.Context, client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]mapiv1beta1.Machine, error) {
	machineList := &mapiv1beta1.MachineList{}

	listOpts := append([]runtimeclient.ListOption{},
		runtimeclient.InNamespace(TestContext.MachineApiNamespace),
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

	if err := client.List(ctx, machineList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineList object: %v", err)
	}

	return machineList.Items, nil
}

// GetMachine get a machine by its name from the default machine API namespace.
func GetMachine(ctx context.Context, client runtimeclient.Client, machineName string) (*mapiv1beta1.Machine, error) {
	machine := &mapiv1beta1.Machine{}
	if err := client.Get(ctx, runtimeclient.ObjectKey{Namespace: TestContext.MachineApiNamespace, Name: machineName}, machine); err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %v", err)
	}
	return machine, nil
}

// DeleteObjectsByLabels list all objects of a given kind by labels and deletes them.
// Currently supported kinds:
// - caov1beta1.MachineAutoscalerList
// - caov1.ClusterAutoscalerList
// - batchv1.JobList
func DeleteObjectsByLabels(ctx context.Context, client runtimeclient.Client, labels map[string]string, list runtime.Object) error {
	if err := client.List(ctx, list, runtimeclient.MatchingLabels(labels)); err != nil {
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
		if err := client.Delete(ctx, obj, &runtimeclient.DeleteOptions{
			PropagationPolicy: &cascadeDelete,
		}); err != nil {
			return fmt.Errorf("error deleting object: %v", err)
		}
	}

	return nil
}

// FilterRunningMachines returns a slice of only those Machines in the input
// that are in the "Running" phase.
func FilterRunningMachines(machines []mapiv1beta1.Machine) []*mapiv1beta1.Machine {
	// TODO(bison): This function should probably take a slice of pointers, but
	// GetMachines() doesn't return that for whatever reason now.
	var result []*mapiv1beta1.Machine

	for i, m := range machines {
		if m.Status.Phase != nil && *m.Status.Phase == MachinePhaseRunning {
			result = append(result, &machines[i])
		}
	}

	return result
}

// GetWorkerMachineSets returns the MachineSets that label their Machines with
// the "worker" role.
func GetWorkerMachineSets(c client.Client) ([]*mapiv1beta1.MachineSet, error) {
	machineSets := &mapiv1beta1.MachineSetList{}

	if err := c.List(context.Background(), machineSets); err != nil {
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

// MachineSetParams represents the parameters for creating a new MachineSet
// resource for use in tests.
type MachineSetParams struct {
	Name         string
	Replicas     int32
	Labels       map[string]string
	ProviderSpec *mapiv1beta1.ProviderSpec
}

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
			Namespace: TestContext.MachineApiNamespace,
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

// WaitForMachineSet waits for the all Machines belonging to the named
// MachineSet to enter the "Running" phase, and for all nodes belonging to those
// Machines to be ready.
func WaitForMachineSet(c client.Client, name string) {
	machineSet, err := GetMachineSet(context.Background(), c, name)
	Expect(err).ToNot(HaveOccurred())

	selector := machineSet.Spec.Selector

	Eventually(func() error {
		machines, err := GetMachines(context.Background(), c, &selector)
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
func WaitForMachineSetDelete(c client.Client, machineSet *mapiv1beta1.MachineSet) {
	Eventually(func() bool {
		selector := machineSet.Spec.Selector

		machines, err := GetMachines(context.Background(), c, &selector)
		if err != nil || len(machines) != 0 {
			return false // Still have Machines, or other error.
		}

		err = c.Get(context.Background(), client.ObjectKey{
			Name:      machineSet.GetName(),
			Namespace: machineSet.GetNamespace(),
		}, &mapiv1beta1.MachineSet{})

		if !apierrors.IsNotFound(err) {
			return false // MachineSet not deleted, or other error.
		}

		return true // MachineSet and Machines were deleted.
	}, WaitLong, RetryMedium).Should(BeTrue())
}

// WaitForMachineDelete polls until the given Machines are not found.
func WaitForMachineDelete(c client.Client, machines ...*mapiv1beta1.Machine) {
	Eventually(func() bool {
		for _, m := range machines {
			err := c.Get(context.Background(), client.ObjectKey{
				Name:      m.GetName(),
				Namespace: m.GetNamespace(),
			}, &mapiv1beta1.Machine{})

			if !apierrors.IsNotFound(err) {
				return false // Not deleted, or other error.
			}
		}

		return true // Everything was deleted.
	}, WaitLong, RetryMedium).Should(BeTrue())
}
