package mapi

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	gotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	framework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var (
	cl client.Client
)

var _ = Describe("[sig-cluster-lifecycle] Machine API GCP MachineSet", framework.LabelMAPI, framework.LabelDisruptive, Ordered, func() {
	var (
		mapiMachineSet *mapiv1.MachineSet
		ctx            context.Context
		platform       configv1.PlatformType
		err            error
	)

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

		if CurrentSpecReport().Failed() && mapiMachineSet != nil {
			logMachineSetFailureDiagnostics(ctx, cl, mapiMachineSet)
		}
		// Clean up MAPI MachineSets
		if mapiMachineSet != nil {
			err := framework.DeleteMachineSets(cl, mapiMachineSet)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete MAPI MachineSet")
			framework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
		}
	})

	It("should have all Shielded VM options disabled when using nonUefi image", framework.LabelPeriodic, func() {
		// Get MAPI machineset parameters
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 0)

		// Override the name to include testcaseid 83064
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-83064-" + uuid.New().String()[0:5]

		// The Marketplace RHCOS 4.8 image is intentionally non-UEFI.
		nonUefiImage := "projects/redhat-marketplace-public/global/images/redhat-coreos-ocp-48-x86-64-202210040145"

		// Unmarshal the provider spec to modify it
		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set the specific image
		providerSpec.Disks[0].Image = nonUefiImage
		// Ensure the MachineSet controller, rather than inherited configuration or
		// admission defaulting, is responsible for setting the disabled policies.
		providerSpec.ShieldedInstanceConfig = mapiv1.GCPShieldedInstanceConfig{}

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with Red Hat CoreOS(nonUefi) image")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for the MachineSet controller to disable all Shielded VM options")
		Eventually(func(g Gomega) {
			current := &mapiv1.MachineSet{}
			g.Expect(cl.Get(ctx, client.ObjectKeyFromObject(mapiMachineSet), current)).To(Succeed())

			actual := &mapiv1.GCPMachineProviderSpec{}
			g.Expect(json.Unmarshal(current.Spec.Template.Spec.ProviderSpec.Value.Raw, actual)).To(Succeed())
			g.Expect(actual.ShieldedInstanceConfig).To(Equal(mapiv1.GCPShieldedInstanceConfig{
				SecureBoot:                       mapiv1.SecureBootPolicyDisabled,
				VirtualizedTrustedPlatformModule: mapiv1.VirtualizedTrustedPlatformModulePolicyDisabled,
				IntegrityMonitoring:              mapiv1.IntegrityMonitoringPolicyDisabled,
			}))
		}, framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "MachineSet template should have all Shielded VM options disabled")

		klog.Infof("Successfully verified that MachineSet %q has all Shielded VM options disabled", mapiMachineSet.Name)
	})

	// Test for provisioningModel: Spot
	It("should provision Spot instance with provisioningModel: Spot successfully", framework.LabelPeriodic, func() {
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
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)

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
	It("should reject when both preemptible: true and provisioningModel: Spot are set", framework.LabelPeriodic, func() {
		By("Building MachineSet parameters from existing cluster")

		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 0)

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
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)
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
		Expect(err).To(MatchError(And(ContainSubstring("preemptible"), ContainSubstring("provisioningModel"))), "Error should mention the conflicting fields")
		// Set mapiMachineSet to nil since creation failed
		mapiMachineSet = nil

		klog.Infof("Successfully verified that webhook rejects both preemptible and provisioningModel being set together")
	})
	// Test: webhook should allow MachineSet update when preemptible is set and provisioningModel is not set
	It("should allow MachineSet update to set preemptible when provisioningModel is not set", framework.LabelPeriodic, func() {
		By("Building MachineSet parameters from existing cluster")

		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 0)

		// Override the name to include testcaseid 85973
		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-85973-update-" + uuid.New().String()[0:5]

		By("Creating initial MachineSet with provisioningModel: Spot and 0 replicas")

		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Set provisioningModel to Spot initially
		providerSpec.ProvisioningModel = ptr.To(mapiv1.GCPSpotInstance)
		providerSpec.OnHostMaintenance = mapiv1.TerminateHostMaintenanceType

		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

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

// logMachineSetFailureDiagnostics emits the resources that explain a failed GCP
// MachineSet reconciliation before AfterEach deletes them. ReconcileError events
// and GCP provider status conditions contain controller and Compute API errors that
// would otherwise be lost during cleanup.
func logMachineSetFailureDiagnostics(ctx context.Context, c client.Client, machineSet *mapiv1.MachineSet) {
	currentMachineSet := &mapiv1.MachineSet{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(machineSet), currentMachineSet); err != nil {
		klog.Errorf("failure diagnostics: get MachineSet %s/%s: %v", machineSet.Namespace, machineSet.Name, err)
		return
	}

	klog.Errorf("failure diagnostics: MachineSet %s/%s status: %+v", currentMachineSet.Namespace, currentMachineSet.Name, currentMachineSet.Status)

	machines, err := framework.GetMachinesFromMachineSet(ctx, c, currentMachineSet)
	if err != nil {
		klog.Errorf("failure diagnostics: list Machines for MachineSet %s/%s: %v", currentMachineSet.Namespace, currentMachineSet.Name, err)
		return
	}

	involvedObjects := map[string]struct{}{currentMachineSet.Name: {}}
	for _, machine := range machines {
		involvedObjects[machine.Name] = struct{}{}
		klog.Errorf("failure diagnostics: Machine %s/%s status: %+v", machine.Namespace, machine.Name, machine.Status)

		if machine.Status.ProviderStatus == nil {
			continue
		}

		providerStatus := &mapiv1.GCPMachineProviderStatus{}
		if err := json.Unmarshal(machine.Status.ProviderStatus.Raw, providerStatus); err != nil {
			klog.Errorf("failure diagnostics: decode GCP provider status for Machine %s/%s: %v; raw status: %s", machine.Namespace, machine.Name, err, machine.Status.ProviderStatus.Raw)
			continue
		}

		klog.Errorf("failure diagnostics: Machine %s/%s GCP provider status: %+v", machine.Namespace, machine.Name, *providerStatus)
	}

	events := &corev1.EventList{}
	if err := c.List(ctx, events, client.InNamespace(currentMachineSet.Namespace)); err != nil {
		klog.Errorf("failure diagnostics: list events in namespace %s: %v", currentMachineSet.Namespace, err)
		return
	}

	for _, event := range events.Items {
		if _, ok := involvedObjects[event.InvolvedObject.Name]; !ok {
			continue
		}

		klog.Errorf("failure diagnostics: event for %s %s/%s: type=%s reason=%s message=%s", event.InvolvedObject.Kind, event.Namespace, event.InvolvedObject.Name, event.Type, event.Reason, event.Message)
	}
}
