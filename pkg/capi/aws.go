package capi

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

const (
	awsMachineTemplateName = "aws-machine-template"
	infrastructureName     = "cluster"
	infraAPIVersion        = "infrastructure.cluster.x-k8s.io/v1beta1"
)

var _ = Describe("Cluster API AWS MachineSet", framework.LabelCAPI, framework.LabelDisruptive, Ordered, func() {
	var (
		cl                      client.Client
		ctx                     = context.Background()
		platform                configv1.PlatformType
		clusterName             string
		oc                      *gatherer.CLI
		awsMachineTemplate      *awsv1.AWSMachineTemplate
		machineSetParams        framework.CAPIMachineSetParams
		machineSet              *clusterv1.MachineSet
		mapiDefaultProviderSpec *mapiv1.AWSMachineProviderConfig
		err                     error
	)

	BeforeAll(func() {
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred(), "Failed to GetConfig")

		cl, err = client.New(cfg, client.Options{})
		Expect(err).ToNot(HaveOccurred(), "Failed to create Kubernetes client for test")

		infra := &configv1.Infrastructure{}
		infraName := client.ObjectKey{
			Name: infrastructureName,
		}
		Expect(cl.Get(ctx, infraName, infra)).To(Succeed(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.PlatformStatus).ToNot(BeNil(), "expected the infrastructure Status.PlatformStatus to not be nil")
		clusterName = infra.Status.InfrastructureName
		platform = infra.Status.PlatformStatus.Type
		if platform != configv1.AWSPlatformType {
			Skip("Skipping AWS E2E tests")
		}
		oc, err = framework.NewCLI()
		Expect(err).ToNot(HaveOccurred(), "Failed to new CLI")
		framework.SkipIfNotTechPreviewNoUpgrade(oc, cl)
		mapiDefaultProviderSpec = getDefaultAWSMAPIProviderSpec(cl)
		machineSetParams = framework.NewCAPIMachineSetParams(
			"aws-machineset",
			clusterName,
			mapiDefaultProviderSpec.Placement.AvailabilityZone,
			1,
			corev1.ObjectReference{
				Kind:       "AWSMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       awsMachineTemplateName,
			},
		)
		framework.CreateCoreCluster(ctx, cl, clusterName, "AWSCluster")
	})

	AfterEach(func() {
		framework.DeleteCAPIMachineSets(ctx, cl, machineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, cl, machineSet)
		framework.DeleteObjects(ctx, cl, awsMachineTemplate)
	})

	//huliu-OCP-51071 - [CAPI] Create machineset with CAPI on aws
	It("should be able to run a machine with a default provider spec", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-51071", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//huliu-OCP-75395 - [CAPI] AWS Placement group support.
	It("should be able to run a machine with cluster placement group", func() {
		awsClient := framework.NewAwsClient(framework.GetCredentialsFromCluster(oc))
		placementGroupName := clusterName + "pgcluster"
		placementGroupID, err := awsClient.CreatePlacementGroup(placementGroupName, "cluster")
		Expect(err).ToNot(HaveOccurred(), "Failed to create placementgroup")
		Expect(placementGroupID).ToNot(Equal(""), "expected the placementGroupID to not be empty string")
		DeferCleanup(func() {
			_, err = awsClient.DeletePlacementGroup(placementGroupName)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete placementgroup")
		})

		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.PlacementGroupName = placementGroupName
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-75395", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//huliu-OCP-75396 - [CAPI] Creating machines using KMS keys from AWS.
	It("should be able to run a machine using KMS keys", framework.LabelQEOnly, func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awskmsClient := framework.NewAwsKmsClient(framework.GetCredentialsFromCluster(oc))
		key, err := awskmsClient.CreateKey(infrastructureName + " key 75396")
		if err != nil {
			Skip("Create key failed, skip the cases!!")
		}
		defer func() {
			err := awskmsClient.DeleteKey(key)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete the key")
		}()

		encryptBool := true
		awsMachineTemplate.Spec.Template.Spec.NonRootVolumes = []awsv1.Volume{
			{
				DeviceName:    "/dev/xvda",
				Size:          140,
				Type:          awsv1.VolumeTypeIO1,
				IOPS:          5000,
				Encrypted:     &encryptBool,
				EncryptionKey: key,
			},
		}
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-75396", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//OCP-78677 - [CAPI] Dedicated tenancy should be exposed on aws providerspec.
	It("should be able to run a machine with dedicated instance", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.Tenancy = "dedicated"
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-78677", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//huliu-OCP-75662 - [CAPI] AWS Machine API Support of more than one block device.
	It("should be able to run a machine with more than one block device", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.NonRootVolumes = []awsv1.Volume{
			{
				DeviceName: "/dev/xvda",
				Size:       120,
				Type:       awsv1.VolumeTypeGP3,
				IOPS:       5000,
				Encrypted:  ptr.To(true),
			},
			{
				DeviceName: "/dev/sdf",
				Size:       120,
				Type:       awsv1.VolumeTypeGP2,
				Encrypted:  ptr.To(false),
			},
		}
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-75662", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//huliu-OCP-75663 - [CAPI] User defined tags can be applied to AWS EC2 Instances.
	It("should be able to run a machine with user defined tags", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.AdditionalTags = map[string]string{
			"adminContact": "qe",
			"costCenter":   "1981",
			"customTag":    "test",
			"Email":        "qe@redhat.com",
		}
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-75663", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//OCP-76794 - [CAPI] Support AWS capacity-reservations in CAPA.
	It("should be able to run a machine with capacity-reservations", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		By("Access AWS to create CapacityReservation")
		awsClient := framework.NewAwsClient(framework.GetCredentialsFromCluster(oc))
		capacityReservationID, err := awsClient.CreateCapacityReservation(mapiDefaultProviderSpec.InstanceType, "Linux/UNIX", mapiDefaultProviderSpec.Placement.AvailabilityZone, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(capacityReservationID).ToNot(Equal(""))

		DeferCleanup(func() {
			_, err := awsClient.CancelCapacityReservation(capacityReservationID)
			Expect(err).ToNot(HaveOccurred(), "Failed to cancel capacityreservation")
		})
		awsMachineTemplate.Spec.Template.Spec.CapacityReservationID = &capacityReservationID
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-76794", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//OCP-81293 - [CAPI][AWS] Support AWS EFA network interface type in CAPI.
	It("should be able to run a machine with EFA network interface type", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.InstanceType = "m5dn.24xlarge"
		awsMachineTemplate.Spec.Template.Spec.NetworkInterfaceType = awsv1.NetworkInterfaceTypeEFAWithENAInterface
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-81293", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//OCP-79026 - [CAPI] Spot instance can be created successfully with CAPI on aws.
	It("should be able to run a machine with SpotMarketOptions", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.SpotMarketOptions = &awsv1.SpotMarketOptions{}
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-79026", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})
})

func getDefaultAWSMAPIProviderSpec(cl client.Client) *mapiv1.AWSMachineProviderConfig {
	machineSetList := &mapiv1.MachineSetList{}

	Eventually(func() error {
		return cl.List(framework.GetContext(), machineSetList, client.InNamespace(framework.MachineAPINamespace))
	}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "it should be able to list the MAPI machinesets")
	Expect(machineSetList.Items).ToNot(HaveLen(0), "expected the MAPI machinesets to be present")

	machineSet := &machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil(), "expected the MAPI machinesets ProviderSpec value to not be nil")

	providerSpec := &mapiv1.AWSMachineProviderConfig{}
	Expect(yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "it should be able to unmarshal the raw yaml into providerSpec")

	klog.Infof("Getting from machineset %v", machineSet.Name)

	return providerSpec
}

func newAWSMachineTemplate(name string, mapiProviderSpec *mapiv1.AWSMachineProviderConfig) *awsv1.AWSMachineTemplate {
	By("Creating AWS machine template")

	Expect(mapiProviderSpec).ToNot(BeNil(), "expected the mapi ProviderSpec to not be nil")
	Expect(mapiProviderSpec.IAMInstanceProfile).ToNot(BeNil(), "expected the mapi IAMInstanceProfile to not be nil")
	Expect(mapiProviderSpec.IAMInstanceProfile.ID).ToNot(BeNil(), "expected the mapi IAMInstanceProfile.ID to not be nil")
	Expect(mapiProviderSpec.InstanceType).ToNot(BeEmpty(), "expected the mapi InstanceType to not be empty")
	Expect(mapiProviderSpec.Placement.AvailabilityZone).ToNot(BeEmpty(), "expected the mapi Placement.AvailabilityZone to not be empty")
	Expect(mapiProviderSpec.AMI.ID).ToNot(BeNil(), "expected the mapi AMI.ID to not be nil")
	Expect(mapiProviderSpec.SecurityGroups).ToNot(HaveLen(0), "expected the mapi SecurityGroups to be present")
	Expect(mapiProviderSpec.SecurityGroups[0].Filters).ToNot(HaveLen(0), "expected the mapi SecurityGroups[0].Filters to be present")
	Expect(mapiProviderSpec.SecurityGroups[0].Filters[0].Values).ToNot(HaveLen(0), "expected the mapi SecurityGroups[0].Filters[0].Values to be present")

	var subnet awsv1.AWSResourceReference

	if len(mapiProviderSpec.Subnet.Filters) == 0 {
		subnet = awsv1.AWSResourceReference{
			ID: mapiProviderSpec.Subnet.ID,
		}
	} else {
		subnet = awsv1.AWSResourceReference{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.Subnet.Filters[0].Values,
				},
			},
		}
	}

	ami := awsv1.AMIReference{
		ID: mapiProviderSpec.AMI.ID,
	}
	ignition := &awsv1.Ignition{
		Version:     "3.4",
		StorageType: awsv1.IgnitionStorageTypeOptionUnencryptedUserData,
	}
	additionalSecurityGroups := []awsv1.AWSResourceReference{
		{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.SecurityGroups[0].Filters[0].Values,
				},
			},
		},
		{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.SecurityGroups[1].Filters[0].Values,
				},
			},
		},
	}

	awsMachineSpec := awsv1.AWSMachineSpec{
		IAMInstanceProfile:       *mapiProviderSpec.IAMInstanceProfile.ID,
		InstanceType:             mapiProviderSpec.InstanceType,
		AMI:                      ami,
		Ignition:                 ignition,
		Subnet:                   &subnet,
		AdditionalSecurityGroups: additionalSecurityGroups,
	}

	awsMachineTemplate := &awsv1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: framework.ClusterAPINamespace,
		},
		Spec: awsv1.AWSMachineTemplateSpec{
			Template: awsv1.AWSMachineTemplateResource{
				Spec: awsMachineSpec,
			},
		},
	}

	return awsMachineTemplate
}
