package machinehealthcheck

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:MachineHealthCheck] MachineHealthCheck", func() {
	var client client.Client

	var machineSet *mapiv1beta1.MachineSet
	var machinehealthcheck *v1beta1.MachineHealthCheck
	var maxUnhealthy = 3
	const expectedReplicas = 5

	const E2EConditionType = "MachineHealthCheckE2E"

	nodeCondition := corev1.NodeCondition{
		Type:               E2EConditionType,
		Status:             corev1.ConditionTrue,
		LastHeartbeatTime:  metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             "E2E",
		Message:            "MachineHealthCheck E2E tests",
	}

	BeforeEach(func() {
		var err error

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		machineSetParams := framework.BuildMachineSetParams(client, expectedReplicas)

		By("Creating a new MachineSet")
		machineSet, err = framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())

		framework.WaitForMachineSet(client, machineSet.GetName())
	})

	AfterEach(func() {
		By("Deleting the MachineHealthCheck resource")
		Expect(client.Delete(context.Background(), machinehealthcheck)).ToNot(HaveOccurred())

		By("Deleting the new MachineSet")
		Expect(client.Delete(context.Background(), machineSet)).ToNot(HaveOccurred())

		framework.WaitForMachineSetDelete(client, machineSet)
	})

	It("should remediate unhealthy nodes", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).ToNot(BeEmpty())

		By("Setting unhealthy conditions on machine nodes, but not exceding maxUnhealthy threshold")
		unhealthyMachines, healthyMachines := machines[:maxUnhealthy-1], machines[maxUnhealthy-1:]
		for _, machine := range unhealthyMachines {
			node, err := framework.GetNodeForMachine(client, machine)
			Expect(err).ToNot(HaveOccurred())
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).ToNot(HaveOccurred())
		}

		By("Creating a MachineHealthCheck resource")
		mhcParams := framework.MachineHealthCheckParams{
			Name:   machineSet.Name,
			Labels: machineSet.Labels,
			Conditions: []mapiv1beta1.UnhealthyCondition{
				{
					Type:    E2EConditionType,
					Status:  corev1.ConditionTrue,
					Timeout: "1s",
				},
			},
			MaxUnhealthy: &maxUnhealthy,
		}

		machinehealthcheck, err = framework.CreateMHC(client, mhcParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(machinehealthcheck).ToNot(BeNil())

		By("Waiting for each unhealthy machine to be deleted")
		framework.WaitForMachinesDeleted(client, unhealthyMachines...)

		By("Waiting for MachineDeleted event from MachineHealthCheck for each unhealthy machine")
		for _, machine := range unhealthyMachines {
			Expect(framework.WaitForEvent(client, "Machine", machine.Name, "MachineDeleted")).ToNot(HaveOccurred())
		}

		By("Ensure none of the healthy machines were deleted")
		allMachines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(allMachines).ToNot(BeEmpty())
		Expect(framework.MachinesPresent(allMachines, healthyMachines...)).To(BeTrue())

		By("Verifying the MachineSet recovers")
		framework.WaitForMachineSet(client, machineSet.GetName())
	})

	It("should not remediate larger number of unhealthy machines then maxUnhealthy", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).ToNot(BeEmpty())

		By("Setting unhealthy conditions on machine nodes, but exceding maxUnhealthy threshold")
		unhealthyMachines := machines[:maxUnhealthy+1]
		for _, machine := range unhealthyMachines {
			node, err := framework.GetNodeForMachine(client, machine)
			Expect(err).ToNot(HaveOccurred())
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).ToNot(HaveOccurred())
		}

		By("Creating a MachineHealthCheck resource")
		mhcParams := framework.MachineHealthCheckParams{
			Name:   machineSet.Name,
			Labels: machineSet.Labels,
			Conditions: []mapiv1beta1.UnhealthyCondition{
				{
					Type:    E2EConditionType,
					Status:  corev1.ConditionTrue,
					Timeout: "1s",
				},
			},
			MaxUnhealthy: &maxUnhealthy,
		}

		machinehealthcheck, err = framework.CreateMHC(client, mhcParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(machinehealthcheck).ToNot(BeNil())

		By("Waiting for RemediationRestricted event from MachineHealthCheck")
		Expect(framework.WaitForEvent(client, "MachineHealthCheck", machineSet.Name, "RemediationRestricted")).ToNot(HaveOccurred())

		By("Ensuring none of the machines were deleted")
		allMachines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(allMachines).ToNot(BeEmpty())
		Expect(framework.MachinesPresent(allMachines, machines...)).To(BeTrue())
	})
})
