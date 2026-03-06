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
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
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
		oc             *gatherer.CLI
		clusterName    string
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

		oc, err = framework.NewCLI()
		Expect(err).ToNot(HaveOccurred(), "Failed to create CLI")

		// Get infrastructure name
		infra := &configv1.Infrastructure{}
		infraName := client.ObjectKey{Name: "cluster"}
		Expect(cl.Get(ctx, infraName, infra)).To(Succeed(), "Failed to get cluster infrastructure object")
		clusterName = infra.Status.InfrastructureName
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

	// getMachineProviderSpec is a helper function to get the AWSMachineProviderConfig from a MachineSet
	getMachineProviderSpec := func(machineSet *mapiv1.MachineSet) *mapiv1.AWSMachineProviderConfig {
		// Get the machines created by this MachineSet
		machines, err := framework.GetMachinesFromMachineSet(ctx, cl, machineSet)
		Expect(err).ToNot(HaveOccurred(), "Getting machines from MachineSet should succeed")
		Expect(machines).ToNot(BeEmpty(), "MachineSet should have at least 1 machine")

		// Get the first machine and unmarshal its provider spec
		machine := machines[0]
		machineProviderSpec := &mapiv1.AWSMachineProviderConfig{}

		By(fmt.Sprintf("Getting machine %q created by MachineSet %q", machine.Name, machineSet.Name))
		Expect(machine.Spec.ProviderSpec.Value).ToNot(BeNil(), "Machine provider spec value should be set")
		Expect(machine.Spec.ProviderSpec.Value.Raw).ToNot(BeEmpty(), "Machine provider spec raw payload should be set")
		Expect(json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, machineProviderSpec)).To(Succeed(), "Should be able to unmarshal machine provider spec")

		return machineProviderSpec
	}

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

		machineProviderSpec := getMachineProviderSpec(mapiMachineSet)

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

		klog.Infof("Successfully verified that MachineSet %q has user defined tags applied", mapiMachineSet.Name)
	})

	//OCP-48594 - [MAPI][AWS] Support AWS EFA network interface type in MAPI.
	It("should be able to run a machine with EFA network interface type", framework.LabelPeriodic, func() {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name with random suffix to avoid conflicts
		machineSetParams.Name = "machineset-efa-" + uuid.New().String()[0:5]
		machineSetParams.Labels[framework.MachineSetKey] = machineSetParams.Name

		providerSpec := &mapiv1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// c5n.9xlarge with EFA may not be available in all regions
		if providerSpec.Placement.Region != "us-east-2" && providerSpec.Placement.Region != "us-west-2" {
			Skip("c5n.9xlarge instances with EFA support may not be available in all regions, limiting this test to us-east-2 and us-west-2")
		}

		// Set instance type and network interface type for EFA
		providerSpec.InstanceType = "c5n.9xlarge"
		providerSpec.NetworkInterfaceType = mapiv1.AWSEFANetworkInterfaceType

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with EFA network interface type")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that machine has EFA network interface type applied")

		machineProviderSpec := getMachineProviderSpec(mapiMachineSet)

		// Verify the instance type and network interface type
		Expect(machineProviderSpec.InstanceType).To(Equal("c5n.9xlarge"), "Machine should have instance type c5n.9xlarge")
		Expect(machineProviderSpec.NetworkInterfaceType).To(Equal(mapiv1.AWSEFANetworkInterfaceType), "Machine should have EFA network interface type")

		klog.Infof("Successfully verified that MachineSet %q has EFA network interface type applied", mapiMachineSet.Name)
	})

	//huliu-OCP-64909 - [MAPI] AWS Placement group support.
	It("should be able to run a machine with cluster placement group", framework.LabelPeriodic, func() {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name with random suffix to avoid conflicts
		machineSetParams.Name = "machineset-pg-" + uuid.New().String()[0:5]
		machineSetParams.Labels[framework.MachineSetKey] = machineSetParams.Name

		providerSpec := &mapiv1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Create AWS client and placement group
		awsClient := framework.NewAwsClient(framework.GetCredentialsFromClusterCtx(ctx, oc))
		placementGroupName := clusterName + "-pgcluster"
		_, err := awsClient.CreatePlacementGroup(placementGroupName, "cluster")
		Expect(err).ToNot(HaveOccurred(), "Failed to create placement group %q", placementGroupName)

		DeferCleanup(func() {
			Eventually(func() error {
				_, err = awsClient.DeletePlacementGroup(placementGroupName)
				return err
			}, framework.WaitShort, framework.RetryMedium).Should(Succeed(), "Failed to delete placement group after retries")
		})

		// Set placement group name in provider spec
		providerSpec.PlacementGroupName = placementGroupName

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with placement group")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that machine has placement group applied")

		machineProviderSpec := getMachineProviderSpec(mapiMachineSet)

		// Verify the placement group name
		Expect(machineProviderSpec.PlacementGroupName).To(Equal(placementGroupName), "Machine should have placement group name set to %s", placementGroupName)

		klog.Infof("Successfully verified that MachineSet %q has placement group %q applied", mapiMachineSet.Name, placementGroupName)
	})

	//huliu-OCP-32122 - [MAPI] AWS Machine API Support of more than one block device.
	It("should be able to run a machine with more than one block device", framework.LabelPeriodic, func() {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name with random suffix to avoid conflicts
		machineSetParams.Name = "machineset-blkdev-" + uuid.New().String()[0:5]
		machineSetParams.Labels[framework.MachineSetKey] = machineSetParams.Name

		providerSpec := &mapiv1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Configure multiple block devices
		providerSpec.BlockDevices = []mapiv1.BlockDeviceMappingSpec{
			{
				DeviceName: ptr.To("/dev/xvda"),
				EBS: &mapiv1.EBSBlockDeviceSpec{
					VolumeSize: ptr.To(int64(120)),
					VolumeType: ptr.To("gp3"),
					Iops:       ptr.To(int64(5000)),
					Encrypted:  ptr.To(true),
				},
			},
			{
				DeviceName: ptr.To("/dev/sdf"),
				EBS: &mapiv1.EBSBlockDeviceSpec{
					VolumeSize: ptr.To(int64(120)),
					VolumeType: ptr.To("gp2"),
					Encrypted:  ptr.To(false),
				},
			},
		}

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with multiple block devices")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that machine has multiple block devices applied")

		machineProviderSpec := getMachineProviderSpec(mapiMachineSet)

		// Verify multiple block devices are configured
		Expect(machineProviderSpec.BlockDevices).To(HaveLen(2), "Machine should have exactly 2 block devices")
		Expect(machineProviderSpec.BlockDevices[0].DeviceName).To(Equal(ptr.To("/dev/xvda")), "First block device should be /dev/xvda")
		Expect(machineProviderSpec.BlockDevices[1].DeviceName).To(Equal(ptr.To("/dev/sdf")), "Second block device should be /dev/sdf")

		klog.Infof("Successfully verified that MachineSet %q has multiple block devices applied", mapiMachineSet.Name)
	})

	//huliu-OCP-37915 - [MAPI] Creating machines using KMS keys from AWS.
	It("should be able to run a machine using KMS keys", framework.LabelQEOnly, framework.LabelPeriodic, func() {
		machineSetParams := framework.BuildMachineSetParams(ctx, cl, 1)

		// Override the name with random suffix to avoid conflicts
		machineSetParams.Name = "machineset-kms-" + uuid.New().String()[0:5]
		machineSetParams.Labels[framework.MachineSetKey] = machineSetParams.Name

		providerSpec := &mapiv1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "Should be able to unmarshal provider spec")

		// Create KMS key
		awskmsClient := framework.NewAwsKmsClient(framework.GetCredentialsFromClusterCtx(ctx, oc))

		key, err := awskmsClient.CreateKey(clusterName + " key 75396")
		if err != nil {
			Skip("Create key failed, skip the cases!!")
		}

		DeferCleanup(func() {
			err := awskmsClient.DeleteKey(key)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete the key")
		})

		// Configure block device with KMS encryption
		providerSpec.BlockDevices = []mapiv1.BlockDeviceMappingSpec{
			{
				DeviceName: ptr.To("/dev/xvda"),
				EBS: &mapiv1.EBSBlockDeviceSpec{
					VolumeSize: ptr.To(int64(140)),
					VolumeType: ptr.To("io1"),
					Iops:       ptr.To(int64(5000)),
					Encrypted:  ptr.To(true),
					KMSKey: mapiv1.AWSResourceReference{
						ARN: &key,
					},
				},
			},
		}

		// Marshal back to raw bytes using JSON
		rawProviderSpec, err := json.Marshal(providerSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to marshal provider spec")

		machineSetParams.ProviderSpec.Value = &runtime.RawExtension{
			Raw: rawProviderSpec,
		}

		By("Creating a new MachineSet with KMS encrypted volume")

		mapiMachineSet, err = framework.CreateMachineSet(cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

		By("Waiting for MachineSet to have running machines")
		framework.WaitForMachineSet(ctx, cl, mapiMachineSet.GetName())

		By("Verifying that machine has KMS key applied")

		machineProviderSpec := getMachineProviderSpec(mapiMachineSet)

		// Verify the KMS key is set
		Expect(machineProviderSpec.BlockDevices).ToNot(BeEmpty(), "Machine should have at least one block device")
		Expect(machineProviderSpec.BlockDevices[0].EBS).ToNot(BeNil(), "First block device should have EBS configuration")
		Expect(machineProviderSpec.BlockDevices[0].EBS.KMSKey.ARN).To(Equal(&key), "EBS volume should use the specified KMS key")

		klog.Infof("Successfully verified that MachineSet %q has KMS key applied", mapiMachineSet.Name)
	})
})
