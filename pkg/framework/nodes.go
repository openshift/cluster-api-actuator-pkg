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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddNodeCondition adds a condition in the given Node's status.
func AddNodeCondition(c runtimeclient.Client, node *corev1.Node, cond corev1.NodeCondition) error {
	nodeCopy := node.DeepCopy()
	nodeCopy.Status.Conditions = append(nodeCopy.Status.Conditions, cond)

	return c.Status().Patch(context.Background(), nodeCopy, runtimeclient.MergeFrom(node))
}

// FilterReadyNodes filters the list of nodes and returns a list with ready nodes.
func FilterReadyNodes(nodes []corev1.Node) []corev1.Node {
	var readyNodes []corev1.Node

	for _, n := range nodes {
		if IsNodeReady(&n) {
			readyNodes = append(readyNodes, n)
		}
	}

	return readyNodes
}

// FilterSchedulableNodes filters the list of nodes and returns a list with schedulable nodes.
func FilterSchedulableNodes(nodes []corev1.Node) []corev1.Node {
	var schedulableNodes []corev1.Node

	for _, n := range nodes {
		if IsNodeSchedulable(&n) {
			schedulableNodes = append(schedulableNodes, n)
		}
	}

	return schedulableNodes
}

// GetNodes gets a list of nodes from a running cluster
// Optionaly, labels may be used to constrain listed nodes.
func GetNodes(c runtimeclient.Client, selectors ...*metav1.LabelSelector) ([]corev1.Node, error) {
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

	if err := c.List(context.TODO(), &nodeList, listOpts...); err != nil {
		return nil, fmt.Errorf("error querying api for nodeList object: %w", err)
	}

	return nodeList.Items, nil
}

// GetNodesFromMachineSet returns an array of nodes backed by machines owned by a given machineSet.
func GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet *machinev1.MachineSet) ([]*corev1.Node, error) {
	machines, err := GetMachinesFromMachineSet(ctx, client, machineSet)
	if err != nil {
		return nil, fmt.Errorf("error calling getMachinesFromMachineSet %w", err)
	}

	var nodes []*corev1.Node

	for key := range machines {
		node, err := GetNodeForMachine(ctx, client, machines[key])
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
func GetNodeForMachine(ctx context.Context, c runtimeclient.Client, m *machinev1.Machine) (*corev1.Node, error) {
	if m.Status.NodeRef == nil {
		return nil, fmt.Errorf("%s: machine has no NodeRef", m.Name)
	}

	node := &corev1.Node{}
	nodeName := runtimeclient.ObjectKey{Name: m.Status.NodeRef.Name}

	if err := c.Get(ctx, nodeName, node); err != nil {
		return nil, err
	}

	return node, nil
}

// GetCAPINodeForMachine retrieves the node backing the given Machine.
func GetCAPINodeForMachine(ctx context.Context, c runtimeclient.Client, m *clusterv1.Machine) (*corev1.Node, error) {
	if m.Status.NodeRef == nil {
		return nil, fmt.Errorf("%s: machine has no NodeRef", m.Name)
	}

	node := &corev1.Node{}
	nodeName := runtimeclient.ObjectKey{Name: m.Status.NodeRef.Name}

	if err := c.Get(ctx, nodeName, node); err != nil {
		return nil, err
	}

	return node, nil
}

// GetReadyAndSchedulableNodes returns all the nodes that have the Ready condition and can schedule workloads.
func GetReadyAndSchedulableNodes(c runtimeclient.Client) ([]corev1.Node, error) {
	nodes, err := GetNodes(c)
	if err != nil {
		return nodes, err
	}

	nodes = FilterReadyNodes(nodes)
	nodes = FilterSchedulableNodes(nodes)

	return nodes, nil
}

// GetWorkerNodes returns all nodes with the nodeWorkerRoleLabel label.
func GetWorkerNodes(c runtimeclient.Client) ([]corev1.Node, error) {
	workerNodes := &corev1.NodeList{}
	if err := c.List(context.TODO(), workerNodes,
		runtimeclient.InNamespace(MachineAPINamespace),
		runtimeclient.MatchingLabels(map[string]string{WorkerNodeRoleLabel: ""}),
	); err != nil {
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

// IsNodeSchedulable returns true is the given node can schedule workloads.
func IsNodeSchedulable(node *corev1.Node) bool {
	return !node.Spec.Unschedulable
}

// NodesAreReady returns true if an array of nodes are all ready.
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

func VerifyNodeDraining(ctx context.Context, client runtimeclient.Client, targetMachine *machinev1.Machine, rc *corev1.ReplicationController) (string, error) {
	endTime := time.Now().Add(WaitLong)

	var drainedNodeName string

	err := wait.PollUntilContextTimeout(ctx, RetryMedium, WaitLong, true, func(ctx context.Context) (bool, error) {
		machine := machinev1.Machine{}

		key := types.NamespacedName{
			Namespace: targetMachine.Namespace,
			Name:      targetMachine.Name,
		}
		if err := client.Get(ctx, key, &machine); err != nil {
			klog.Errorf("Error querying api machine %q object: %v, retrying...", targetMachine.Name, err)
			return false, nil
		}

		if machine.Status.NodeRef == nil || machine.Status.NodeRef.Kind != "Node" {
			klog.Errorf("Machine %q not linked to a node", machine.Name)
			return false, nil
		}

		drainedNodeName = machine.Status.NodeRef.Name
		node := corev1.Node{}

		if err := client.Get(ctx, types.NamespacedName{Name: drainedNodeName}, &node); err != nil {
			klog.Errorf("Error querying api node %q object: %v, retrying...", drainedNodeName, err)
			return false, nil
		}

		if !node.Spec.Unschedulable {
			klog.Errorf("Node %q is expected to be marked as unschedulable, it is not", node.Name)
			return false, nil
		}

		klog.Infof("[remaining %s] Node %q is mark unschedulable as expected", remainingTime(endTime), node.Name)

		pods := corev1.PodList{}
		if err := client.List(ctx, &pods, runtimeclient.MatchingLabels(rc.Spec.Selector)); err != nil {
			klog.Errorf("Error querying api for Pods object: %v, retrying...", err)
			return false, nil
		}

		podCounter := 0

		for _, pod := range pods.Items {
			if pod.Spec.NodeName == machine.Status.NodeRef.Name && pod.DeletionTimestamp.IsZero() {
				podCounter++
			}
		}

		klog.Infof("[remaining %s] Have %v pods scheduled to node %q", remainingTime(endTime), podCounter, machine.Status.NodeRef.Name)

		// Verify we have enough pods running as well
		rcObj := corev1.ReplicationController{}
		key = types.NamespacedName{
			Namespace: rc.Namespace,
			Name:      rc.Name,
		}

		if err := client.Get(ctx, key, &rcObj); err != nil {
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

func WaitUntilAllRCPodsAreReady(ctx context.Context, client runtimeclient.Client, rc *corev1.ReplicationController) error {
	endTime := time.Now().Add(WaitLong)
	err := wait.PollUntilContextTimeout(ctx, RetryMedium, WaitLong, true, func(ctx context.Context) (bool, error) {
		rcObj := corev1.ReplicationController{}
		key := types.NamespacedName{
			Namespace: rc.Namespace,
			Name:      rc.Name,
		}

		if err := client.Get(ctx, key, &rcObj); err != nil {
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
	if err := client.List(ctx, &pods, runtimeclient.MatchingLabels(rc.Spec.Selector)); err != nil {
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

func WaitUntilNodeDoesNotExists(ctx context.Context, client runtimeclient.Client, nodeName string) error {
	endTime := time.Now().Add(WaitLong)

	return wait.PollUntilContextTimeout(ctx, RetryMedium, WaitLong, true, func(ctx context.Context) (bool, error) {
		node := corev1.Node{}
		key := types.NamespacedName{
			Name: nodeName,
		}

		err := client.Get(ctx, key, &node)
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
func WaitUntilAllNodesAreReady(ctx context.Context, client runtimeclient.Client) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, PollNodesReadyTimeout, true, func(ctx context.Context) (bool, error) {
		nodeList := corev1.NodeList{}
		if err := client.List(ctx, &nodeList); err != nil {
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
	return time.Until(t).Round(time.Second)
}
