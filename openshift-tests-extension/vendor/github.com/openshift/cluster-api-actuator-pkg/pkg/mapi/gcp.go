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
})
