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

var _ = Describe("[sig-cluster-lifecycle] Machine API AWS MachineSet", framework.LabelMAPI, framework.LabelDisruptive, Ordered, ContinueOnFailure, func() {
	var (
		mapiMachineSet *mapiv1.MachineSet
		ctx            context.Context
		platform       configv1.PlatformType
		err            error
		cl             client.Client
	)

	BeforeAll(func() {
		cl, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(cl)

		ctx = framework.GetContext()

		platform, err = framework.GetPlatform(ctx, cl)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")

		if platform != configv1.AWSPlatformType {
			Skip("Skipping AWS E2E tests")
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

	//huliu-OCP-41469 - [MAPI] User defined tags can be applied to AWS EC2 Instances.
	It("should be able to run a machine with user defined tags", framework.LabelPeriodic, func() {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name with random suffix to avoid conflicts
		machineSetParams.Name = "machineset-tags-" + uuid.New().String()[0:5]
		machineSetParams.Labels[framework.MachineSetKey] = machineSetParams.Name

		providerSpec := &mapiv1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Add user defined tags
		providerSpec.Tags = append(providerSpec.Tags, []mapiv1.TagSpecification{
			{Name: "adminContact", Value: "qe"},
			{Name: "costCenter", Value: "1981"},
			{Name: "customTag", Value: "test"},
			{Name: "Email", Value: "qe@redhat.com"},
		}...)

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with user defined tags")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that machine has user defined tags applied")
		// Get the machines created by this MachineSet
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, mapiMachineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).To(HaveLen(1), "MachineSet should have exactly 1 machine")

		// Get the first machine created by this MachineSet and verify its provider spec
		machine := machines[0]
		machineProviderSpec := &mapiv1.AWSMachineProviderConfig{}

		By(fmt.Sprintf("Getting machine %q created by MachineSet %q", machine.Name, mapiMachineSet.Name))
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, machineProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		// Verify the tags are present in the machine's provider spec
		expectedTags := map[string]string{
			"adminContact": "qe",
			"costCenter":   "1981",
			"customTag":    "test",
			"Email":        "qe@redhat.com",
		}

		actualTags := make(map[string]string)
		for _, tag := range machineProviderSpec.Tags {
			actualTags[tag.Name] = tag.Value
		}

		for name, value := range expectedTags {
			Expect(actualTags).To(HaveKeyWithValue(name, value), "Machine should have tag %s=%s", name, value)
		}

		klog.Infof("Successfully verified that machine %q has user defined tags applied", machine.Name)
	})
})
