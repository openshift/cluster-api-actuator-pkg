package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FilterRunningMachines returns a slice of only those Machines in the input
// that are in the "Running" phase.
func FilterRunningMachines(machines []*mapiv1beta1.Machine) []*mapiv1beta1.Machine {
	var result []*mapiv1beta1.Machine

	for i, m := range machines {
		if m.Status.Phase != nil && *m.Status.Phase == MachinePhaseRunning {
			result = append(result, machines[i])
		}
	}

	return result
}

// GetMachine get a machine by its name from the default machine API namespace.
func GetMachine(c client.Client, name string) (*mapiv1beta1.Machine, error) {
	machine := &mapiv1beta1.Machine{}
	key := client.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := c.Get(context.Background(), key, machine); err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %w", err)
	}

	return machine, nil
}

// GetMachines gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachines(client runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]*mapiv1beta1.Machine, error) {
	machineList := &mapiv1beta1.MachineList{}

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

	if err := client.List(context.Background(), machineList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineList object: %w", err)
	}

	var machines []*mapiv1beta1.Machine

	for i := range machineList.Items {
		machines = append(machines, &machineList.Items[i])
	}

	return machines, nil
}

// GetMachineFromNode returns the Machine associated with the given node.
func GetMachineFromNode(client runtimeclient.Client, node *corev1.Node) (*mapiv1beta1.Machine, error) {
	machineNamespaceKey, ok := node.Annotations[MachineAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("node %q does not have a MachineAnnotationKey %q",
			node.Name, MachineAnnotationKey)
	}
	namespace, machineName, err := cache.SplitMetaNamespaceKey(machineNamespaceKey)
	if err != nil {
		return nil, fmt.Errorf("machine annotation format is incorrect %v: %v",
			machineNamespaceKey, err)
	}

	if namespace != MachineAPINamespace {
		return nil, fmt.Errorf("Machine %q is forbidden to live outside of default %v namespace",
			machineNamespaceKey, MachineAPINamespace)
	}

	machine, err := GetMachine(client, machineName)
	if err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %w", err)
	}

	return machine, nil
}

// DeleteMachine deletes a specific machine and returns an error otherwise
func DeleteMachine(client runtimeclient.Client, machine *mapiv1beta1.Machine) error {
	return wait.PollImmediate(1*time.Second, time.Minute, func() (bool, error) {
		if err := client.Delete(context.TODO(), machine); err != nil {
			klog.Errorf("Error querying api for machine object %q: %v, retrying...", machine.Name, err)
			return false, err
		}
		return true, nil
	})
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
