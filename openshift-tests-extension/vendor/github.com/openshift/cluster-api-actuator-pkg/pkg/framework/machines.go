package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FilterMachines returns a slice of only those Machines in the input that are
// in the requested phase.
func FilterMachines(machines []*machinev1.Machine, phase string) []*machinev1.Machine {
	var result []*machinev1.Machine

	for i, m := range machines {
		if m.Status.Phase != nil && *m.Status.Phase == phase {
			result = append(result, machines[i])
		}
	}

	return result
}

// FilterRunningMachines returns a slice of only those Machines in the input
// that are in the "Running" phase.
func FilterRunningMachines(machines []*machinev1.Machine) []*machinev1.Machine {
	return FilterMachines(machines, MachinePhaseRunning)
}

// GetMachine get a machine by its name from the default machine API namespace.
func GetMachine(c runtimeclient.Client, name string) (*machinev1.Machine, error) {
	machine := &machinev1.Machine{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := c.Get(context.Background(), key, machine); err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %w", err)
	}

	return machine, nil
}

// MachinesPresent search for each provided machine in `machines` argument in the predefined `existingMachines` list
// and returns true when all of them were found.
func MachinesPresent(existingMachines []*machinev1.Machine, machines ...*machinev1.Machine) bool {
	if len(existingMachines) < len(machines) {
		return false
	}

	existingMachinesMap := map[types.UID]struct{}{}
	for _, existing := range existingMachines {
		existingMachinesMap[existing.UID] = struct{}{}
	}

	for _, machine := range machines {
		if _, found := existingMachinesMap[machine.UID]; !found {
			return false
		}
	}

	return true
}

// GetMachines gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachines(ctx context.Context, client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]*machinev1.Machine, error) {
	machineList := &machinev1.MachineList{}

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

	if err := client.List(ctx, machineList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineList object: %w", err)
	}

	var machines []*machinev1.Machine

	for i := range machineList.Items {
		machines = append(machines, &machineList.Items[i])
	}

	return machines, nil
}

// GetMachineFromNode returns the Machine associated with the given node.
func GetMachineFromNode(client runtimeclient.Client, node *corev1.Node) (*machinev1.Machine, error) {
	machineNamespaceKey, ok := node.Annotations[MachineAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("node %q does not have a MachineAnnotationKey %q",
			node.Name, MachineAnnotationKey)
	}

	namespace, machineName, err := cache.SplitMetaNamespaceKey(machineNamespaceKey)
	if err != nil {
		return nil, fmt.Errorf("machine annotation format is incorrect %v: %w",
			machineNamespaceKey, err)
	}

	if namespace != MachineAPINamespace {
		return nil, fmt.Errorf("machine %q is forbidden to live outside of default %v namespace",
			machineNamespaceKey, MachineAPINamespace)
	}

	machine, err := GetMachine(client, machineName)
	if err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %w", err)
	}

	return machine, nil
}

// DeleteMachines deletes the specified machines and returns an error on failure.
func DeleteMachines(ctx context.Context, client runtimeclient.Client, machines ...*machinev1.Machine) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, time.Minute, true, func(ctx context.Context) (bool, error) {
		for _, machine := range machines {
			err := client.Delete(ctx, machine)
			if err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("Error querying api for machine object %q: %v, retrying...", machine.Name, err)
				return false, err
			}
		}

		return true, nil
	})
}

// WaitForMachinesDeleted polls until the given Machines are not found.
func WaitForMachinesDeleted(c runtimeclient.Client, machines ...*machinev1.Machine) {
	Eventually(func() bool {
		for _, m := range machines {
			if err := c.Get(context.Background(), runtimeclient.ObjectKey{
				Name:      m.GetName(),
				Namespace: m.GetNamespace(),
			}, &machinev1.Machine{}); !apierrors.IsNotFound(err) {
				return false // Not deleted, or other error.
			}
		}

		return true // Everything was deleted.
	}, WaitLong, RetryMedium).Should(BeTrue(), "error encountered while waiting for Machines to be deleted.")
}
