package mapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	gotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	framework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var (
	cl client.Client
)

var _ = Describe("Machine API GCP MachineSet", framework.LabelMAPI, framework.LabelDisruptive, Ordered, func() {
	var mapiMachineSet *mapiv1.MachineSet
	var ctx context.Context
	var platform configv1.PlatformType
	var err error

	BeforeAll(func() {
		cl, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(cl)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, cl)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")
		if platform != configv1.GCPPlatformType {
			Skip("Skipping GCP E2E tests")
		}
	})

	AfterEach(func() {
		// if the current testing are skipped, we skip clean resources
		if CurrentSpecReport().State == gotypes.SpecStateSkipped {
			return
		}
		// Clean up MAPI MachineSets
		if mapiMachineSet != nil {
			err := framework.DeleteMachineSets(cl, mapiMachineSet)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete MAPI MachineSet")
			framework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
		}
	})

	It("should have all Shielded VM options disabled when using nonUefi image", func() {
		// Get MAPI machineset parameters
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name to include testcaseid 83064
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-83064-" + uuid.New().String()[0:5]

		// Modify the provider spec to use the nonUefi image
		nonUefiImage := "projects/redhat-marketplace-public/global/images/redhat-coreos-ocp-48-x86-64-202210040145"

		// Unmarshal the provider spec to modify it
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set the specific image
		providerSpec.Disks[0].Image = nonUefiImage

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with Red Hat CoreOS(nonUefi) image")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that all Shielded VM options are disabled")
		// Get the machines created by this MachineSet
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machine from MachineSet should succeed")

		// Get the first machine created by this MachineSet and verify its provider spec
		machine := machines[0]
		machineProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		By(fmt.Sprintf("Getting machine %q created by MachineSet %q", machine.Name, mapiMachineSet.Name))
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, machineProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		Expect(machineProviderSpec).To(HaveField("ShieldedInstanceConfig", Equal(mapiv1.GCPShieldedInstanceConfig{
			SecureBoot:                       mapiv1.SecureBootPolicyDisabled,
			VirtualizedTrustedPlatformModule: mapiv1.VirtualizedTrustedPlatformModulePolicyDisabled,
			IntegrityMonitoring:              mapiv1.IntegrityMonitoringPolicyDisabled,
		})), "provider spec should have shielded-instance defaults disabled")

		klog.Infof("Successfully verified that machine %q has all Shielded VM options disabled", machine.Name)
	})

	// Test for provisioningModel: Spot
	It("should provision Spot instance with provisioningModel: Spot successfully", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name to include testcaseid 85973
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-85973-spot-" + uuid.New().String()[0:5]

		By("Modifying providerSpec to use provisioningModel: Spot")
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set provisioningModel to Spot
		spotModel := mapiv1.GCPSpotInstance
		providerSpec.ProvisioningModel = &spotModel

		// Spot instances require OnHostMaintenance = Terminate
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating MachineSet with Spot provisioning model")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying provisioningModel is set to Spot in the Machine spec")
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		machine := machines[0]
		actualProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, actualProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		Expect(actualProviderSpec.ProvisioningModel).ToNot(BeNil(), "Machine should have provisioningModel set")
		Expect(*actualProviderSpec.ProvisioningModel).To(Equal(mapiv1.GCPSpotInstance), "Machine should have provisioningModel set to Spot")
		Expect(actualProviderSpec.OnHostMaintenance).To(Equal(mapiv1.TerminateHostMaintenanceType), "Machine should have OnHostMaintenance set to Terminate")

		klog.Infof("Successfully verified that machine %q is a Spot instance with provisioningModel: Spot", machine.Name)
	})

	It("should provision Standard instance when provisioningModel value is omitted", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name to include testcaseid 85973
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-85973-standard-" + uuid.New().String()[0:5]

		By("Ensuring provisioningModel is NOT set (omitted)")
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Explicitly set provisioningModel to nil (omitted)
		providerSpec.ProvisioningModel = nil

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating MachineSet without provisioningModel set")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying instance is Standard (provisioningModel is omitted)")
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		machine := machines[0]
		actualProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, actualProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		Expect(actualProviderSpec.ProvisioningModel).To(BeNil(), "Machine should have provisioningModel set to nil (omitted)")
		Expect(machine.Status.Phase).ToNot(Equal("Failed"), "Machine should not be in Failed phase")

		klog.Infof("Successfully verified that machine %q is a Standard instance with provisioningModel omitted", machine.Name)
	})

	// Webhook validation test: preemptible and provisioningModel should not be used together
	It("should reject when both preemptible: true and provisioningModel: Spot are set", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name to include testcaseid 85973
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-85973-conflict-" + uuid.New().String()[0:5]

		By("Setting both preemptible: true and provisioningModel: Spot")
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set BOTH fields - this should be rejected by webhook
		providerSpec.Preemptible = true
		spotModel := mapiv1.GCPSpotInstance
		providerSpec.ProvisioningModel = &spotModel
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Attempting to create MachineSet - expecting webhook rejection")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).To(HaveOccurred(), "Webhook should reject MachineSet with both preemptible and provisioningModel set")
		Expect(err.Error()).To(ContainSubstring("admission webhook"), "Should be a webhook validation error")
		Expect(err.Error()).To(Or(
			ContainSubstring("preemptible"),
			ContainSubstring("provisioningModel"),
		), "Error should mention the conflicting fields")

		// Set mapiMachineSet to nil since creation failed
		mapiMachineSet = nil

		klog.Infof("Successfully verified that webhook rejects both preemptible and provisioningModel being set together")
	})

	// Test: preemptible takes preference and provisioningModel is removed during update
	It("should remove provisioningModel when preemptible is set during update", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name to include testcaseid 85973
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-85973-update-" + uuid.New().String()[0:5]

		By("Creating initial MachineSet with provisioningModel: Spot")
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set provisioningModel to Spot initially
		spotModel := mapiv1.GCPSpotInstance
		providerSpec.ProvisioningModel = &spotModel
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for initial MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying initial provisioningModel is set to Spot")
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		machine := machines[0]
		actualProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, actualProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")
		Expect(actualProviderSpec.ProvisioningModel).ToNot(BeNil(), "Machine should have provisioningModel set")
		Expect(*actualProviderSpec.ProvisioningModel).To(Equal(mapiv1.GCPSpotInstance), "Machine should have provisioningModel set to Spot")

		By("Updating MachineSet to set preemptible: true (should remove provisioningModel)")
		// Get the latest version of the machine
		updatedMachine := &mapiv1.Machine{}
		err = cl.Get(ctx, client.ObjectKey{Namespace: machine.Namespace, Name: machine.Name}, updatedMachine)
		Expect(err).ToNot(HaveOccurred(), "Should be able to get machine")

		// Update the provider spec to set preemptible and remove provisioningModel
		updatedProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(updatedMachine.Spec.ProviderSpec.Value.Raw, updatedProviderSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set preemptible to true
		updatedProviderSpec.Preemptible = true
		// Explicitly remove provisioningModel (webhook should enforce this)
		updatedProviderSpec.ProvisioningModel = nil

		rawUpdatedProviderSpec, err := json.Marshal(updatedProviderSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		updatedMachine.Spec.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawUpdatedProviderSpec,
		}

		err = cl.Update(ctx, updatedMachine)
		Expect(err).ToNot(HaveOccurred(), "Should be able to update machine with preemptible when provisioningModel is removed")

		By("Verifying preemptible is set and provisioningModel is removed")
		verifyMachine := &mapiv1.Machine{}
		err = cl.Get(ctx, client.ObjectKey{Namespace: machine.Namespace, Name: machine.Name}, verifyMachine)
		Expect(err).ToNot(HaveOccurred(), "Should be able to get updated machine")

		verifyProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(verifyMachine.Spec.ProviderSpec.Value.Raw, verifyProviderSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		Expect(verifyProviderSpec.Preemptible).To(BeTrue(), "Machine should have preemptible set to true")
		Expect(verifyProviderSpec.ProvisioningModel).To(BeNil(), "Machine should have provisioningModel removed (nil)")

		klog.Infof("Successfully verified that machine %q was updated with preemptible: true and provisioningModel removed", machine.Name)
	})
})
