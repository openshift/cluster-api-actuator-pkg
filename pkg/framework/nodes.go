package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddNodeCondition adds a condition in the given Node's status.
func AddNodeCondition(c client.Client, node *corev1.Node, cond corev1.NodeCondition) error {
	nodeCopy := node.DeepCopy()
	nodeCopy.Status.Conditions = append(nodeCopy.Status.Conditions, cond)

	return c.Status().Patch(context.Background(), nodeCopy, client.MergeFrom(node))
}

// FilterReadyNodes fileter the list of nodes and returns the list with ready nodes
func FilterReadyNodes(nodes []corev1.Node) []corev1.Node {
	var readyNodes []corev1.Node
	for _, n := range nodes {
		if IsNodeReady(&n) {
			readyNodes = append(readyNodes, n)
		}
	}
	return readyNodes
}

// GetNodes gets a list of nodes from a running cluster
// Optionaly, labels may be used to constrain listed nodes.
func GetNodes(c client.Client, selectors ...*metav1.LabelSelector) ([]corev1.Node, error) {
	var listOpts []client.ListOption

	nodeList := corev1.NodeList{}

	for _, selector := range selectors {
		s, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, err
		}

		listOpts = append(listOpts,
			client.MatchingLabelsSelector{Selector: s},
		)
	}

	if err := c.List(context.TODO(), &nodeList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for nodeList object: %w", err)
	}

	return nodeList.Items, nil
}

// GetNodesFromMachineSet returns an array of nodes backed by machines owned by a given machineSet
func GetNodesFromMachineSet(client runtimeclient.Client, machineSet *machinev1.MachineSet) ([]*corev1.Node, error) {
	machines, err := GetMachinesFromMachineSet(client, machineSet)
	if err != nil {
		return nil, fmt.Errorf("error calling getMachinesFromMachineSet %w", err)
	}

	var nodes []*corev1.Node
	for key := range machines {
		node, err := GetNodeForMachine(client, machines[key])
		if apierrors.IsNotFound(err) {
			// We don't care about not found errors.
			// Callers should account for the number of nodes being correct or not.
			klog.Infof("No Node object found for machine %s", machines[key].Name)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error getting node from machine %q: %w", machines[key].Name, err)
		}
		nodes = append(nodes, node)
	}
	klog.Infof("MachineSet %q have %d nodes", machineSet.Name, len(nodes))
	return nodes, nil
}

// GetNodeForMachine retrieves the node backing the given Machine.
func GetNodeForMachine(c client.Client, m *machinev1.Machine) (*corev1.Node, error) {
	if m.Status.NodeRef == nil {
		return nil, fmt.Errorf("%s: machine has no NodeRef", m.Name)
	}

	node := &corev1.Node{}
	nodeName := client.ObjectKey{Name: m.Status.NodeRef.Name}

	if err := c.Get(context.Background(), nodeName, node); err != nil {
		return nil, err
	}

	return node, nil
}

// GetWorkerNodes returns all nodes with the nodeWorkerRoleLabel label
func GetWorkerNodes(c client.Client) ([]corev1.Node, error) {
	workerNodes := &corev1.NodeList{}
	err := c.List(context.TODO(), workerNodes,
		client.InNamespace(MachineAPINamespace),
		client.MatchingLabels(map[string]string{WorkerNodeRoleLabel: ""}),
	)

	if err != nil {
		return nil, err
	}

	return workerNodes.Items, nil
}

// IsNodeReady returns true if the given node is ready.
func IsNodeReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// NodesAreReady returns true if an array of nodes are all ready
func NodesAreReady(nodes []*corev1.Node) bool {
	// All nodes needs to be ready
	for key := range nodes {
		if !IsNodeReady(nodes[key]) {
			klog.Errorf("Node %q is not ready. Conditions are: %v", nodes[key].Name, nodes[key].Status.Conditions)
			return false
		}
		klog.Infof("Node %q is ready. Conditions are: %v", nodes[key].Name, nodes[key].Status.Conditions)
	}
	return true
}

func VerifyNodeDraining(client runtimeclient.Client, targetMachine *machinev1.Machine, rc *corev1.ReplicationController) (string, error) {
	endTime := time.Now().Add(time.Duration(WaitLong))
	var drainedNodeName string
	err := wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		machine := machinev1.Machine{}

		key := types.NamespacedName{
			Namespace: targetMachine.Namespace,
			Name:      targetMachine.Name,
		}
		if err := client.Get(context.TODO(), key, &machine); err != nil {
			klog.Errorf("Error querying api machine %q object: %v, retrying...", targetMachine.Name, err)
			return false, nil
		}
		if machine.Status.NodeRef == nil || machine.Status.NodeRef.Kind != "Node" {
			klog.Errorf("Machine %q not linked to a node", machine.Name)
			return false, nil
		}

		drainedNodeName = machine.Status.NodeRef.Name
		node := corev1.Node{}

		if err := client.Get(context.TODO(), types.NamespacedName{Name: drainedNodeName}, &node); err != nil {
			klog.Errorf("Error querying api node %q object: %v, retrying...", drainedNodeName, err)
			return false, nil
		}

		if !node.Spec.Unschedulable {
			klog.Errorf("Node %q is expected to be marked as unschedulable, it is not", node.Name)
			return false, nil
		}

		klog.Infof("[remaining %s] Node %q is mark unschedulable as expected", remainingTime(endTime), node.Name)

		pods := corev1.PodList{}
		if err := client.List(context.TODO(), &pods, runtimeclient.MatchingLabels(rc.Spec.Selector)); err != nil {
			klog.Errorf("Error querying api for Pods object: %v, retrying...", err)
			return false, nil
		}

		podCounter := 0
		for _, pod := range pods.Items {
			if pod.Spec.NodeName != machine.Status.NodeRef.Name {
				continue
			}
			if !pod.DeletionTimestamp.IsZero() {
				continue
			}
			podCounter++
		}

		klog.Infof("[remaining %s] Have %v pods scheduled to node %q", remainingTime(endTime), podCounter, machine.Status.NodeRef.Name)

		// Verify we have enough pods running as well
		rcObj := corev1.ReplicationController{}
		key = types.NamespacedName{
			Namespace: rc.Namespace,
			Name:      rc.Name,
		}
		if err := client.Get(context.TODO(), key, &rcObj); err != nil {
			klog.Errorf("Error querying api RC %q object: %v, retrying...", rc.Name, err)
			return false, nil
		}

		// The point of the test is to make sure majority of the pods are rescheduled
		// to other nodes. Pod disruption budget makes sure at most one pod
		// owned by the RC is not Ready. So no need to test it. Though, useful to have it printed.
		klog.Infof("[remaining %s] RC ReadyReplicas: %v, Replicas: %v", remainingTime(endTime), rcObj.Status.ReadyReplicas, rcObj.Status.Replicas)

		// This makes sure at most one replica is not ready
		if rcObj.Status.Replicas-rcObj.Status.ReadyReplicas > 1 {
			return false, fmt.Errorf("pod disruption budget not respected, node was not properly drained")
		}

		// Depends on timing though a machine can be deleted even before there is only
		// one pod left on the node (that is being evicted).
		if podCounter > 2 {
			klog.Infof("[remaining %s] Expecting at most 2 pods to be scheduled to drained node %q, got %v", remainingTime(endTime), machine.Status.NodeRef.Name, podCounter)
			return false, nil
		}

		klog.Infof("[remaining %s] Expected result: all pods from the RC up to last one or two got scheduled to a different node while respecting PDB", remainingTime(endTime))
		return true, nil
	})

	return drainedNodeName, err
}

func WaitUntilAllRCPodsAreReady(client runtimeclient.Client, rc *corev1.ReplicationController) error {
	endTime := time.Now().Add(time.Duration(WaitLong))
	err := wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		rcObj := corev1.ReplicationController{}
		key := types.NamespacedName{
			Namespace: rc.Namespace,
			Name:      rc.Name,
		}
		if err := client.Get(context.TODO(), key, &rcObj); err != nil {
			klog.Errorf("Error querying api RC %q object: %v, retrying...", rc.Name, err)
			return false, nil
		}
		if rcObj.Status.ReadyReplicas == 0 {
			klog.Infof("[%s remaining] Waiting for at least one RC ready replica, ReadyReplicas: %v, Replicas: %v", remainingTime(endTime), rcObj.Status.ReadyReplicas, rcObj.Status.Replicas)
			return false, nil
		}
		klog.Infof("[%s remaining] Waiting for RC ready replicas, ReadyReplicas: %v, Replicas: %v", remainingTime(endTime), rcObj.Status.ReadyReplicas, rcObj.Status.Replicas)
		return rcObj.Status.Replicas == rcObj.Status.ReadyReplicas, nil
	})

	// Sometimes this will timeout because Status.Replicas !=
	// Status.ReadyReplicas. Print the state of all the pods for
	// debugging purposes so we can distinguish between the cases
	// when it works and those rare cases when it doesn't.
	pods := corev1.PodList{}
	if err := client.List(context.TODO(), &pods, runtimeclient.MatchingLabels(rc.Spec.Selector)); err != nil {
		klog.Errorf("Error listing pods: %v", err)
	} else {
		prettyPrint := func(i interface{}) string {
			s, _ := json.MarshalIndent(i, "", "  ")
			return string(s)
		}
		for i := range pods.Items {
			klog.Infof("POD #%v/%v: %s", i, len(pods.Items), prettyPrint(pods.Items[i]))
		}
	}

	return err
}

func WaitUntilNodeDoesNotExists(client runtimeclient.Client, nodeName string) error {
	endTime := time.Now().Add(time.Duration(WaitLong))
	return wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		node := corev1.Node{}

		key := types.NamespacedName{
			Name: nodeName,
		}
		err := client.Get(context.TODO(), key, &node)
		if err == nil {
			klog.Errorf("Node %q not yet deleted", nodeName)
			return false, nil
		}

		if !strings.Contains(err.Error(), "not found") {
			klog.Errorf("Error querying api node %q object: %v, retrying...", nodeName, err)
			return false, nil
		}

		klog.Infof("[%s remaining] Node %q successfully deleted", remainingTime(endTime), nodeName)
		return true, nil
	})
}

// WaitUntilAllNodesAreReady lists all nodes and waits until they are ready.
func WaitUntilAllNodesAreReady(client runtimeclient.Client) error {
	return wait.PollImmediate(RetryShort, PollNodesReadyTimeout, func() (bool, error) {
		nodeList := corev1.NodeList{}
		if err := client.List(context.TODO(), &nodeList); err != nil {
			klog.Errorf("error querying api for nodeList object: %v, retrying...", err)
			return false, nil
		}
		// All nodes needs to be ready
		for _, node := range nodeList.Items {
			if !IsNodeReady(&node) {
				klog.Errorf("Node %q is not ready", node.Name)
				return false, nil
			}
		}
		return true, nil
	})
}

func remainingTime(t time.Time) time.Duration {
	return t.Sub(time.Now()).Round(time.Second)
}
