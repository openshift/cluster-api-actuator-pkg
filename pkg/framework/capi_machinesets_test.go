package framework

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	machineSet *clusterv1.MachineSet
	machines   []*clusterv1.Machine
}

func (c *machineSetWaitTestClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if machineSet, ok := obj.(*clusterv1.MachineSet); ok {
		*machineSet = *c.machineSet.DeepCopy()
		return nil
	}

	return fmt.Errorf("unexpected object type %T", obj)
}

func (c *machineSetWaitTestClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	machineList, ok := list.(*clusterv1.MachineList)
	if !ok {
		return fmt.Errorf("unexpected list type %T", list)
	}

	for _, machine := range c.machines {
		machineList.Items = append(machineList.Items, *machine.DeepCopy())
	}

	return nil
}
