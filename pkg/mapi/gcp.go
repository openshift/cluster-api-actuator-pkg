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
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
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

	// buildMachineSetParamsAndProviderSpec builds MachineSet parameters and unmarshals the provider spec
	// for use in test cases. It returns the machineSetParams and providerSpec.
	buildMachineSetParamsAndProviderSpec := func(replicas int, nameSuffix string) (framework.MachineSetParams, *mapiv1.GCPMachineProviderSpec) {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, replicas)

		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-" + nameSuffix + "-" + uuid.New().String()[0:5]

		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		return machineSetParams, providerSpec
	}

	// marshalProviderSpec marshals the provider spec and updates the machineSetParams with it.
	marshalProviderSpec := func(machineSetParams *framework.MachineSetParams, providerSpec *mapiv1.GCPMachineProviderSpec) {
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}
	}

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
		machineSetParams, providerSpec := buildMachineSetParamsAndProviderSpec(1, "83064")

		// Modify the provider spec to use the nonUefi image
		nonUefiImage := "projects/redhat-marketplace-public/global/images/redhat-coreos-ocp-48-x86-64-202210040145"

		// Set the specific image
		providerSpec.Disks[0].Image = nonUefiImage

		marshalProviderSpec(&machineSetParams, providerSpec)

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
		machineSetParams, providerSpec := buildMachineSetParamsAndProviderSpec(1, "85973-spot")

		By("Modifying providerSpec to use provisioningModel: Spot")

		// Set provisioningModel to Spot
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)

		// Spot instances require OnHostMaintenance = Terminate
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		marshalProviderSpec(&machineSetParams, providerSpec)

		By("Creating MachineSet with Spot provisioning model")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying machine has interruptible-instance label set")
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		machine := machines[0]
		actualProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, actualProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		Expect(machine.Spec.ObjectMeta.Labels).To(HaveKeyWithValue(machinecontroller.MachineInterruptibleInstanceLabelName, ""), "Machine should have interruptible-instance label set")
		Expect(actualProviderSpec.OnHostMaintenance).To(Equal(mapiv1.TerminateHostMaintenanceType), "Machine should have OnHostMaintenance set to Terminate")

		klog.Infof("Successfully verified that machine %q has interruptible-instance label set", machine.Name)
	})

	// Webhook validation test: preemptible and provisioningModel should not be used together
	It("should reject when both preemptible: true and provisioningModel: Spot are set", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams, providerSpec := buildMachineSetParamsAndProviderSpec(0, "85973-conflict")

		By("Setting both preemptible: true and provisioningModel: Spot")

		// Set BOTH fields - this should be rejected by webhook
		providerSpec.Preemptible = true
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		marshalProviderSpec(&machineSetParams, providerSpec)

		By("Attempting to create MachineSet - expecting webhook rejection")
		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).To(HaveOccurred(), "Webhook should reject MachineSet with both preemptible and provisioningModel set")
		Expect(err.Error()).To(ContainSubstring("admission webhook"), "Should be a webhook validation error")
		Expect(err).To(MatchError(And(ContainSubstring("preemptible"), ContainSubstring("provisioningModel"))), "Error should mention the conflicting fields")
		// Set mapiMachineSet to nil since creation failed
		mapiMachineSet = nil

		klog.Infof("Successfully verified that webhook rejects both preemptible and provisioningModel being set together")
	})
	// Test: webhook should allow MachineSet update when preemptible is set and provisioningModel is not set
	It("should allow MachineSet update to set preemptible when provisioningModel is not set", func() {
		By("Building MachineSet parameters from existing cluster")
		machineSetParams, providerSpec := buildMachineSetParamsAndProviderSpec(0, "85973-update")

		By("Creating initial MachineSet with provisioningModel: Spot and 0 replicas")

		// Set provisioningModel to Spot initially
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		marshalProviderSpec(&machineSetParams, providerSpec)

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Verifying initial MachineSet template has provisioningModel set to Spot")
		initialProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(mapiMachineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, initialProviderSpec)).To(Succeed(), "Should be able to unmarshal provider spec")
		Expect(initialProviderSpec.ProvisioningModel).ToNot(BeNil(), "MachineSet template should have provisioningModel set")
		Expect(*initialProviderSpec.ProvisioningModel).To(Equal(mapiv1.GCPSpotInstance), "MachineSet template should have provisioningModel set to Spot")

		By("Updating MachineSet template to set preemptible: true and ensure provisioningModel is not set to Spot")
		Eventually(komega.Update(mapiMachineSet, func() {
			updatedProviderSpec := &mapiv1.GCPMachineProviderSpec{}
			Expect(json.Unmarshal(mapiMachineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, updatedProviderSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

			// Set preemptible to true
			updatedProviderSpec.Preemptible = true
			// Ensure provisioningModel is not set to Spot (set to nil/omitted)
			updatedProviderSpec.ProvisioningModel = nil

			rawUpdatedProviderSpec, err := json.Marshal(updatedProviderSpec)
			Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

			mapiMachineSet.Spec.Template.Spec.ProviderSpec.Value = &runtime.RawExtension{
				Raw: rawUpdatedProviderSpec,
			}
		})).Should(Succeed(), "Should be able to update MachineSet template with preemptible when provisioningModel is not set to Spot")

		By("Verifying MachineSet template has preemptible set and provisioningModel is not set to Spot")
		verifyMachineSet := &mapiv1.MachineSet{}
		err = cl.Get(ctx, client.ObjectKey{Namespace: mapiMachineSet.Namespace, Name: mapiMachineSet.Name}, verifyMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Should be able to get updated MachineSet")

		verifyProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(verifyMachineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, verifyProviderSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		Expect(verifyProviderSpec.Preemptible).To(BeTrue(), "MachineSet template should have preemptible set to true")
		Expect(verifyProviderSpec.ProvisioningModel).To(BeNil(), "MachineSet template should have provisioningModel not set to Spot (nil/omitted)")

		klog.Infof("Successfully verified that MachineSet %q template was updated with preemptible: true and provisioningModel is not set to Spot", mapiMachineSet.Name)
	})
})
