package framework

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNodeForMachine retrieves the node backing the given Machine.
func GetNodeForMachine(cl client.Client, m *clusterv1.Machine) (*corev1.Node, error) {
	if m.Status.NodeRef == nil {
		return nil, fmt.Errorf("%s: machine has no NodeRef", m.Name)
	}

	node := &corev1.Node{}
	nodeName := client.ObjectKey{Name: m.Status.NodeRef.Name}

	if err := cl.Get(ctx, nodeName, node); err != nil {
		return nil, err
	}

	return node, nil
}

// isNodeReady returns true if the given node is ready.
func isNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
