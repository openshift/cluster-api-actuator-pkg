package framework

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetMachines gets a list of machines from the default cluster API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachines(cl client.Client, selectors ...*metav1.LabelSelector) ([]*clusterv1.Machine, error) {
	machineList := &clusterv1.MachineList{}

	listOpts := append([]client.ListOption{},
		client.InNamespace(CAPINamespace),
	)

	for _, selector := range selectors {
		s, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}

		listOpts = append(listOpts,
			client.MatchingLabelsSelector{Selector: s},
		)
	}

	if err := cl.List(ctx, machineList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for machineList object: %w", err)
	}

	var machines []*clusterv1.Machine

	for i := range machineList.Items {
		machines = append(machines, &machineList.Items[i])
	}

	return machines, nil
}

// FilterRunningMachines returns a slice of only those Machines in the input
// that are in the "Running" phase.
func FilterRunningMachines(machines []*clusterv1.Machine) []*clusterv1.Machine {
	var result []*clusterv1.Machine

	for i, m := range machines {
		if m.Status.Phase == string(clusterv1.MachinePhaseRunning) {
			result = append(result, machines[i])
		}
	}

	return result
}
