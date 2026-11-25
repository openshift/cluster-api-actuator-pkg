package framework

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCheckCAPIMachineSetDeletedWaitsForOwnedNodes(t *testing.T) {
	ctx := context.Background()
	ms := &clusterv1.MachineSet{ObjectMeta: metav1.ObjectMeta{
		Name:      "worker-machineset",
		Namespace: ClusterAPINamespace,
	}}
	ownedNode := corev1.Node{ObjectMeta: metav1.ObjectMeta{
		Name: "worker-node",
		Annotations: map[string]string{
			clusterv1.OwnerKindAnnotation:        "MachineSet",
			clusterv1.OwnerNameAnnotation:        ms.Name,
			clusterv1.ClusterNamespaceAnnotation: ms.Namespace,
		},
	}}
	otherNamespaceNode := *ownedNode.DeepCopy()
	otherNamespaceNode.Name = "other-namespace-node"
	otherNamespaceNode.Annotations[clusterv1.ClusterNamespaceAnnotation] = "other-namespace"
	unrelatedNode := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "control-plane-node"}}

	cl := &machineSetWaitTestClient{nodes: []corev1.Node{unrelatedNode, otherNamespaceNode, ownedNode}}
	err := checkCAPIMachineSetDeleted(ctx, cl, ms)
	if err == nil || !strings.Contains(err.Error(), ownedNode.Name) || !strings.Contains(err.Error(), ms.Name) {
		t.Fatalf("checkCAPIMachineSetDeleted() error = %v, want lingering Node and MachineSet names", err)
	}

	cl.nodes = []corev1.Node{unrelatedNode, otherNamespaceNode}
	if err := checkCAPIMachineSetDeleted(ctx, cl, ms); err != nil {
		t.Fatalf("checkCAPIMachineSetDeleted() after Node deletion = %v, want nil", err)
	}

	listErr := errors.New("Node list unavailable")
	cl.nodeListErr = listErr
	if err := checkCAPIMachineSetDeleted(ctx, cl, ms); !errors.Is(err, listErr) {
		t.Fatalf("checkCAPIMachineSetDeleted() error = %v, want %v", err, listErr)
	}
}

func TestWaitForCAPIMachinesRunningWithRetryReturnsTerminalProvisioningCondition(t *testing.T) {
	gomega.RegisterTestingT(t)

	replicas := int32(1)
	machineSet := &clusterv1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker-machineset",
			Namespace: ClusterAPINamespace,
			UID:       "worker-machineset-uid",
		},
		Spec: clusterv1.MachineSetSpec{Replicas: &replicas},
	}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "worker-machine",
			Namespace: ClusterAPINamespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(machineSet, clusterv1.GroupVersion.WithKind("MachineSet")),
			},
		},
		Status: clusterv1.MachineStatus{
			Phase: string(clusterv1.MachinePhaseProvisioning),
			Conditions: []metav1.Condition{
				{
					Type:    clusterv1.ReadyCondition,
					Status:  metav1.ConditionFalse,
					Reason:  clusterv1.MachineNotReadyReason,
					Message: "Machine is not ready",
				},
				{
					Type:    clusterv1.InfrastructureReadyCondition,
					Status:  metav1.ConditionFalse,
					Reason:  invalidConfigurationReason,
					Message: "unsupported instance type",
				},
			},
		},
	}
	cl := &machineSetWaitTestClient{machineSet: machineSet, machines: []*clusterv1.Machine{machine}}

	err := WaitForCAPIMachinesRunningWithRetry(context.Background(), cl, machineSet.Name, nil)
	if err == nil {
		t.Fatal("WaitForCAPIMachinesRunningWithRetry() error = nil, want terminal provisioning error")
	}

	for _, want := range []string{"worker-machine", "InfrastructureReady", "InvalidConfiguration", "unsupported instance type"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("WaitForCAPIMachinesRunningWithRetry() error = %q, want it to contain %q", err, want)
		}
	}
}

func TestTerminalProvisioningConditionIgnoresTransientNotReady(t *testing.T) {
	machine := &clusterv1.Machine{
		Status: clusterv1.MachineStatus{
			Conditions: []metav1.Condition{{
				Type:    clusterv1.ReadyCondition,
				Status:  metav1.ConditionFalse,
				Reason:  clusterv1.MachineNotReadyReason,
				Message: "Machine is still provisioning",
			}},
		},
	}

	if condition := terminalProvisioningCondition(machine); condition != nil {
		t.Errorf("terminalProvisioningCondition() = %#v, want nil for transient NotReady", condition)
	}
}

type machineSetWaitTestClient struct {
	client.Client
	machineSet  *clusterv1.MachineSet
	machines    []*clusterv1.Machine
	nodes       []corev1.Node
	nodeListErr error
}

func (c *machineSetWaitTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if machineSet, ok := obj.(*clusterv1.MachineSet); ok {
		if c.machineSet == nil {
			return apierrors.NewNotFound(clusterv1.GroupVersion.WithResource("machinesets").GroupResource(), key.Name)
		}

		*machineSet = *c.machineSet.DeepCopy()
		return nil
	}

	return fmt.Errorf("unexpected object type %T", obj)
}

func (c *machineSetWaitTestClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if nodeList, ok := list.(*corev1.NodeList); ok {
		nodeList.Items = c.nodes
		return c.nodeListErr
	}

	machineList, ok := list.(*clusterv1.MachineList)
	if !ok {
		return fmt.Errorf("unexpected list type %T", list)
	}

	for _, machine := range c.machines {
		machineList.Items = append(machineList.Items, *machine.DeepCopy())
	}

	return nil
}
