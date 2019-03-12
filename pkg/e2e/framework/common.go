package framework

import (
	"context"
	"fmt"
	"time"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WorkerRoleLabel = "node-role.kubernetes.io/worker"
	WaitShort       = 1 * time.Minute
	WaitMedium      = 3 * time.Minute
	WaitLong        = 10 * time.Minute
	RetryMedium     = 5 * time.Second
)

// GetNodes gets a list of nodes from a running cluster
// Optionaly, labels may be used to constrain listed nodes.
func GetNodes(client runtimeclient.Client, labels ...map[string]string) ([]corev1.Node, error) {
	nodeList := corev1.NodeList{}
	listOptions := &runtimeclient.ListOptions{}
	if len(labels) > 0 && len(labels[0]) > 0 {
		listOptions.MatchingLabels(labels[0])
	}
	if err := client.List(context.TODO(), listOptions, &nodeList); err != nil {
		return nil, fmt.Errorf("error querying api for nodeList object: %v", err)
	}
	return nodeList.Items, nil
}

// GetMachineSets gets a list of machinesets from the default machine API namespace.
// Optionaly, labels may be used to constrain listed machinesets.
func GetMachineSets(ctx context.Context, client runtimeclient.Client, labels ...map[string]string) ([]mapiv1beta1.MachineSet, error) {
	machineSetList := &mapiv1beta1.MachineSetList{}
	listOptions := runtimeclient.InNamespace(TestContext.MachineApiNamespace)
	if len(labels) > 0 && len(labels[0]) > 0 {
		listOptions.MatchingLabels(labels[0])
	}
	if err := client.List(ctx, listOptions, machineSetList); err != nil {
		return nil, fmt.Errorf("error querying api for machineSetList object: %v", err)
	}
	return machineSetList.Items, nil
}
