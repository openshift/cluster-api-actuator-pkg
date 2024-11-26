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

var _ = Describe("MachineHealthCheck", framework.LabelMachineHealthCheck, framework.LabelDisruptive, func() {
	var client client.Client
	var ctx context.Context

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

		ctx = framework.GetContext()

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "failed to create a new StateGatherer")

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "failed to create a new controller-runtime client")

		machineSetParams := framework.BuildMachineSetParams(ctx, client, expectedReplicas)

		By("Creating a new MachineSet")
		machineSet, err = framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "failed to create a new machineSet resource")

		// Make sure to clean up the resources we created
		DeferCleanup(func() {
			By("Deleting the MachineHealthCheck resource")
			Expect(client.Delete(context.Background(), machinehealthcheck)).To(Succeed(), "failed to delete MHC")

			By("Deleting the new MachineSet")
			Expect(client.Delete(context.Background(), machineSet)).To(Succeed(), "failed to delete machineSet")

			framework.WaitForMachineSetsDeleted(ctx, client, machineSet)
		})

		framework.WaitForMachineSet(ctx, client, machineSet.GetName())
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "failed to gather spec report")
		}
	})

	// Machines required for test: 3
	// Reason: 1 unhealthy, 1 healthy, 1 replacement for the unhealthy
	It("should remediate unhealthy nodes", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(ctx, client, &selector)
		Expect(err).ToNot(HaveOccurred(), "failed to get machines using a selector")
		Expect(machines).ToNot(BeEmpty(), "expected to get a non empty list of machines, got an empty list of machines")

		By("Setting unhealthy conditions on machine nodes, but not exceding maxUnhealthy threshold")
		unhealthyMachines, healthyMachines := machines[:maxUnhealthy], machines[maxUnhealthy:]
		for _, machine := range unhealthyMachines {
			node, err := framework.GetNodeForMachine(ctx, client, machine)
			Expect(err).ToNot(HaveOccurred(), "failed to get a node for a machine")
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).To(Succeed(), "failed to add a condition in a node's status")
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
		Expect(err).ToNot(HaveOccurred(), "failed to create a new MHC resource")
		Expect(machinehealthcheck).ToNot(BeNil(), "expected the new MHC resource to not be nil")

		By("Waiting for each unhealthy machine to be deleted")
		framework.WaitForMachinesDeleted(client, unhealthyMachines...)

		By("Waiting for MachineDeleted event from MachineHealthCheck for each unhealthy machine")
		for _, machine := range unhealthyMachines {
			Expect(framework.WaitForEvent(ctx, client, "Machine", machine.Name, "MachineDeleted")).To(Succeed(), "failed to find event MachineDeleted for machine named %s: %v", machine.GetName(), err)
		}

		By("Ensure none of the healthy machines were deleted")
		allMachines, err := framework.GetMachines(ctx, client, &selector)
		Expect(err).ToNot(HaveOccurred(), "failed to get machines using a selector: %v", err)
		Expect(allMachines).ToNot(BeEmpty(), "expected to get a non empty list of machines, got an empty list of machines")
		Expect(framework.MachinesPresent(allMachines, healthyMachines...)).To(BeTrue(), "expected all machines to be present, but atleast one is not present")

		By("Verifying the MachineSet recovers")
		framework.WaitForMachineSet(ctx, client, machineSet.GetName())
	})

	// Machines required for test: 2
	// Reason: We have two unhealthy machines, but the maxUnhealthy threshold is 1, so the MHC should not remediate.
	It("should not remediate larger number of unhealthy machines then maxUnhealthy", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(ctx, client, &selector)
		Expect(err).ToNot(HaveOccurred(), "failed to get machines using a selector")
		Expect(machines).ToNot(BeEmpty(), "expected to get a non empty list of machines, got an empty list of machines")

		By("Setting unhealthy conditions on machine nodes, but exceding maxUnhealthy threshold")
		unhealthyMachines := machines[:maxUnhealthy+1]
		for _, machine := range unhealthyMachines {
			node, err := framework.GetNodeForMachine(ctx, client, machine)
			Expect(err).ToNot(HaveOccurred(), "failed to get a node for machine")
			Expect(framework.AddNodeCondition(client, node, nodeCondition)).To(Succeed(), "failed to a condition in a node's status")
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
		Expect(err).ToNot(HaveOccurred(), "failed to create a new MHC resource")
		Expect(machinehealthcheck).ToNot(BeNil(), "expected the new MHC resource to not be nil")

		By("Waiting for RemediationRestricted event from MachineHealthCheck")
		Expect(framework.WaitForEvent(ctx, client, "MachineHealthCheck", machineSet.Name, "RemediationRestricted")).To(Succeed(), "failed to find event RemediationRestricted for MHC")

		By("Ensuring none of the machines were deleted")
		allMachines, err := framework.GetMachines(ctx, client, &selector)
		Expect(err).ToNot(HaveOccurred(), "failed to get machines using a selector")
		Expect(machines).ToNot(BeEmpty(), "expected to get a non empty list of machines, got an empty list of machines")
		Expect(framework.MachinesPresent(allMachines, machines...)).To(BeTrue(), "expected all machines to be present, but atleast one is not present")
	})
})
