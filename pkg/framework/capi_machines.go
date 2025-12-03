package framework

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetCAPIMachines gets a list of machines from the default cluster API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetCAPIMachines(ctx context.Context, cl client.Client, selectors ...*metav1.LabelSelector) ([]*clusterv1beta1.Machine, error) {
	machineList := &clusterv1beta1.MachineList{}

	listOpts := append([]client.ListOption{},
		client.InNamespace(ClusterAPINamespace),
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

	var machines []*clusterv1beta1.Machine

	for i := range machineList.Items {
		machines = append(machines, &machineList.Items[i])
	}

	return machines, nil
}

// FilterCAPIMachinesInPhase returns a slice of only those Machines in the input that are in the selected phase.
func FilterCAPIMachinesInPhase(machines []*clusterv1beta1.Machine, machinePhase string) []*clusterv1beta1.Machine {
	var result []*clusterv1beta1.Machine

	for i, m := range machines {
		if m.Status.Phase == machinePhase {
			result = append(result, machines[i])
		}
	}

	return result
}
