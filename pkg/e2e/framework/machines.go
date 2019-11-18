package framework

import (
	"context"
	"fmt"

	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const MachineAnnotationKey = "machine.openshift.io/machine"

// GetMachineFromNode returns the machine referenced by the "controllernode.MachineAnnotationKey" annotation in the given node
func GetMachineFromNode(client runtimeclient.Client, node *corev1.Node) (*mapiv1beta1.Machine, error) {
	machineNamespaceKey, ok := node.Annotations[MachineAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("node %q does not have a MachineAnnotationKey %q", node.Name, MachineAnnotationKey)
	}
	namespace, machineName, err := cache.SplitMetaNamespaceKey(machineNamespaceKey)
	if err != nil {
		return nil, fmt.Errorf("machine annotation format is incorrect %v: %v", machineNamespaceKey, err)
	}

	if namespace != MachineAPINamespace {
		return nil, fmt.Errorf("Machine %q is forbidden to live outside of default %v namespace", machineNamespaceKey, MachineAPINamespace)
	}

	machine, err := GetMachine(context.TODO(), client, machineName)
	if err != nil {
		return nil, fmt.Errorf("error querying api for machine object: %v", err)
	}

	return machine, nil
}
