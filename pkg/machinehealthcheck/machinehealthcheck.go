package machinehealthcheck

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("MachineHealthCheck", framework.LabelMachineHealthChecks, func() {
	var client client.Client

	var gatherer *gatherer.StateGatherer

	var machineSet *machinev1.MachineSet
	var machinehealthcheck *machinev1.MachineHealthCheck
	var maxUnhealthy = 1
	const expectedReplicas = 2

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

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred())

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		machineSetParams := framework.BuildMachineSetParams(client, expectedReplicas)

		By("Creating a new MachineSet")
		machineSet, err = framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())

		framework.WaitForMachineSet(client, machineSet.GetName())
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
		}

		By("Deleting the MachineHealthCheck resource")
		Expect(client.Delete(context.Background(), machinehealthcheck)).To(Succeed())

		By("Deleting the new MachineSet")
		Expect(client.Delete(context.Background(), machineSet)).To(Succeed())

		framework.WaitForMachineSetsDeleted(client, machineSet)
	})

	// Machines required for test: 3
	// Reason: 1 unhealthy, 1 healthy, 1 replacement for the unhealthy
	It("should remediate unhealthy nodes", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).ToNot(BeEmpty())

		By("Setting unhealthy conditions on machine nodes, but not exceding maxUnhealthy threshold")
		unhealthyMachines, healthyMachines := machines[:maxUnhealthy], machines[maxUnhealthy:]
		for _, machine := range unhealthyMachines {
			node, err := framework.GetNodeForMachine(client, machine)
			Expect(err).ToNot(HaveOccurred())
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).To(Succeed())
		}

		By("Creating a MachineHealthCheck resource")
		mhcParams := framework.MachineHealthCheckParams{
			Name:   machineSet.Name,
			Labels: machineSet.Labels,
			Conditions: []machinev1.UnhealthyCondition{
				{
					Type:    E2EConditionType,
					Status:  corev1.ConditionTrue,
					Timeout: metav1.Duration{Duration: time.Second},
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
			Expect(framework.WaitForEvent(client, "Machine", machine.Name, "MachineDeleted")).To(Succeed())
		}

		By("Ensure none of the healthy machines were deleted")
		allMachines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(allMachines).ToNot(BeEmpty())
		Expect(framework.MachinesPresent(allMachines, healthyMachines...)).To(BeTrue())

		By("Verifying the MachineSet recovers")
		framework.WaitForMachineSet(client, machineSet.GetName())
	})

	// Machines required for test: 2
	// Reason: We have two unhealthy machines, but the maxUnhealthy threshold is 1, so the MHC should not remediate.
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
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).To(Succeed())
		}

		By("Creating a MachineHealthCheck resource")
		mhcParams := framework.MachineHealthCheckParams{
			Name:   machineSet.Name,
			Labels: machineSet.Labels,
			Conditions: []machinev1.UnhealthyCondition{
				{
					Type:    E2EConditionType,
					Status:  corev1.ConditionTrue,
					Timeout: metav1.Duration{Duration: time.Second},
				},
			},
			MaxUnhealthy: &maxUnhealthy,
		}

		machinehealthcheck, err = framework.CreateMHC(client, mhcParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(machinehealthcheck).ToNot(BeNil())

		By("Waiting for RemediationRestricted event from MachineHealthCheck")
		Expect(framework.WaitForEvent(client, "MachineHealthCheck", machineSet.Name, "RemediationRestricted")).To(Succeed())

		By("Ensuring none of the machines were deleted")
		allMachines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(allMachines).ToNot(BeEmpty())
		Expect(framework.MachinesPresent(allMachines, machines...)).To(BeTrue())
	})
})
