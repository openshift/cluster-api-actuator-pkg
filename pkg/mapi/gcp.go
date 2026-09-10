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

const (
	coreOSBootImagesNamespace = "openshift-machine-config-operator"
	coreOSBootImagesName      = "coreos-bootimages"
)

// coreOSBootImageStream is a minimal view of the CoreOS stream metadata, holding
// only the GCP image fields this suite needs.
type coreOSBootImageStream struct {
	Architectures map[string]struct {
		Images struct {
			GCP *struct {
				Project string `json:"project"`
				Name    string `json:"name"`
			} `json:"gcp"`
		} `json:"images"`
	} `json:"architectures"`
}

// gcpImagesFromCoreOSBootImages returns every GCP image reference published in the
// coreos-bootimages ConfigMap, across all streams and architectures. A boot image
// resolved by the GCP actuator must be one of these.
func gcpImagesFromCoreOSBootImages(ctx context.Context, cl client.Client) []string {
	cm := &corev1.ConfigMap{}
	Expect(cl.Get(ctx, client.ObjectKey{
		Namespace: coreOSBootImagesNamespace,
		Name:      coreOSBootImagesName,
	}, cm)).To(Succeed(), "Should be able to get the coreos-bootimages ConfigMap")

	rawStreams := map[string]json.RawMessage{}

	if streams, ok := cm.Data["streams"]; ok {
		Expect(json.Unmarshal([]byte(streams), &rawStreams)).To(Succeed(), "Should be able to parse the streams key of the coreos-bootimages ConfigMap")
	} else {
		// Older clusters publish a single stream under the deprecated "stream" key.
		stream, ok := cm.Data["stream"]
		Expect(ok).To(BeTrue(), "coreos-bootimages ConfigMap should have either a streams or a stream key")

		rawStreams["stream"] = json.RawMessage(stream)
	}

	images := []string{}

	for name, raw := range rawStreams {
		stream := &coreOSBootImageStream{}
		Expect(json.Unmarshal(raw, stream)).To(Succeed(), fmt.Sprintf("Should be able to parse stream %q from the coreos-bootimages ConfigMap", name))

		for _, arch := range stream.Architectures {
			if arch.Images.GCP == nil {
				continue
			}

			images = append(images, fmt.Sprintf("projects/%s/global/images/%s", arch.Images.GCP.Project, arch.Images.GCP.Name))
		}
	}

	return images
}

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
		// Clean up MAPI MachineSets
		if mapiMachineSet != nil {
			err := framework.DeleteMachineSets(cl, mapiMachineSet)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete MAPI MachineSet")
			framework.WaitForMachineSetsDeleted(ctx, cl, mapiMachineSet)
		}
	})

	It("should have all Shielded VM options disabled when using nonUefi image", framework.LabelPeriodic, func() {
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

	// Machines whose provider spec omits the boot disk image have it resolved by
	// the GCP actuator at reconcile time, from the coreos-bootimages ConfigMap.
	// Installer-created MachineSets always carry an explicit image, so this is the
	// only test that exercises that path.
	//
	// This requires the MAPI admission webhook to have stopped defaulting the boot
	// disk image; until then the webhook refills the field with its own constant
	// and this test fails.
	It("should resolve the boot disk image when the provider spec omits it", func() {
		By("Building MachineSet parameters from existing cluster")

		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		machineSetParams.Name = infra.Status.InfrastructureName + "-bootimage-" + uuid.New().String()[0:5]

		By("Clearing the boot disk image from the provider spec")

		providerSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")
		Expect(providerSpec.Disks).ToNot(BeEmpty(), "Worker provider spec should define at least one disk")

		providerSpec.Disks[0].Image = ""

		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a MachineSet with no boot disk image")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for the MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying the actuator resolved a boot disk image onto the machine")

		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		machine := machines[0]
		machineProviderSpec := &mapiv1.GCPMachineProviderSpec{}
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, machineProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")
		Expect(machineProviderSpec.Disks).ToNot(BeEmpty(), "Machine provider spec should define at least one disk")

		expectedImages := gcpImagesFromCoreOSBootImages(ctx, cl)
		Expect(expectedImages).ToNot(BeEmpty(), "coreos-bootimages ConfigMap should publish at least one GCP image")

		Expect(machineProviderSpec.Disks[0].Image).To(BeElementOf(expectedImages),
			"boot disk image should have been resolved from the coreos-bootimages ConfigMap")

		klog.Infof("Successfully verified that machine %q resolved boot disk image %q", machine.Name, machineProviderSpec.Disks[0].Image)
	})
})
