package capi

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

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
		machineSet              *clusterv1beta1.MachineSet
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
		if machineSet != nil {
			framework.DeleteCAPIMachineSets(ctx, cl, machineSet)
			framework.WaitForCAPIMachineSetsDeleted(ctx, cl, machineSet)
		}
		if awsMachineTemplate != nil {
			framework.DeleteObjects(ctx, cl, awsMachineTemplate)
		}
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
			Eventually(func() error {
				_, err = awsClient.DeletePlacementGroup(placementGroupName)
				return err
			}, framework.WaitShort, framework.RetryMedium).Should(Succeed(), "Failed to delete placementgroup after retries")
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
		var success bool
		machineSet, awsMachineTemplate, success = createAWSCAPIMachineSetWithRetry(ctx, cl, "aws-machineset-78677", clusterName, mapiDefaultProviderSpec, 4, func(template *awsv1.AWSMachineTemplate, instanceType string) {
			template.Spec.Template.Spec.Tenancy = "dedicated"
		})
		if !success {
			// Resources have been cleaned up during retry, set to nil to avoid duplicate cleanup in AfterEach
			machineSet = nil
			awsMachineTemplate = nil
			Skip("Unable to create machine with dedicated instance after 4 retries due to insufficient capacity")
		}
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
		capacityReservationID, err := awsClient.CreateCapacityReservation(mapiDefaultProviderSpec.InstanceType, "Linux/UNIX", mapiDefaultProviderSpec.Placement.AvailabilityZone, 1, "targeted")
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
		// c5n.9xlarge with EFA may not be available in all regions
		if mapiDefaultProviderSpec.Placement.Region != "us-east-2" && mapiDefaultProviderSpec.Placement.Region != "us-west-2" {
			Skip("c5n.9xlarge instances with EFA support may not be available in all regions, limiting this test to us-east-2 and us-west-2")
		}

		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.InstanceType = "c5n.9xlarge"
		awsMachineTemplate.Spec.Template.Spec.NetworkInterfaceType = awsv1.NetworkInterfaceTypeEFAWithENAInterface
		Expect(cl.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-81293", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)
	})

	//OCP-79026 - [CAPI] Spot instance can be created successfully with CAPI on aws.
	It("should be able to run a machine with SpotMarketOptions", func() {
		var success bool
		machineSet, awsMachineTemplate, success = createAWSCAPIMachineSetWithRetry(ctx, cl, "aws-machineset-79026a", clusterName, mapiDefaultProviderSpec, 4, func(template *awsv1.AWSMachineTemplate, instanceType string) {
			template.Spec.Template.Spec.SpotMarketOptions = &awsv1.SpotMarketOptions{}
		})
		if !success {
			// Resources have been cleaned up during retry, set to nil to avoid duplicate cleanup in AfterEach
			machineSet = nil
			awsMachineTemplate = nil
			Skip("Unable to create machine with SpotMarketOptions after 4 retries due to insufficient capacity")
		}
	})

	//OCP-84243 - [CAPI] AWS capacity reservation preference None should work correctly.
	It("should be able to run a machine with capacity reservation preference None", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.CapacityReservationPreference = awsv1.CapacityReservationPreferenceNone
		Eventually(func() error {
			return cl.Create(ctx, awsMachineTemplate)
		}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-84243-none", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)

		instance := getAWSInstanceConfig(ctx, cl, oc, machineSet)
		Expect(instance.CapacityReservationSpecification.CapacityReservationPreference).To(Equal(ptr.To("none")), fmt.Sprintf("Instance should have capacity reservation preference set to '%s'", "none"))
		Expect(instance.CapacityReservationId).Should(BeNil(), "Instance should not be in any capacity because capacityReservationPreference is None")
	})

	//OCP-84243 - [CAPI] AWS capacity reservation preference CapacityReservationsOnly should work correctly.
	It("should be able to run a machine with capacity reservation preference CapacityReservationsOnly", func() {
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.InstanceType = "m6i.large"
		awsMachineTemplate.Spec.Template.Spec.CapacityReservationPreference = awsv1.CapacityReservationPreferenceOnly

		By("Access AWS to create CapacityReservation")
		awsClient := framework.NewAwsClient(framework.GetCredentialsFromCluster(oc))
		capacityReservationID, err := awsClient.CreateCapacityReservation(awsMachineTemplate.Spec.Template.Spec.InstanceType, "Linux/UNIX", mapiDefaultProviderSpec.Placement.AvailabilityZone, 1, "open")
		Expect(err).ToNot(HaveOccurred())
		Expect(capacityReservationID).ToNot(Equal(""))

		DeferCleanup(func() {
			_, err := awsClient.CancelCapacityReservation(capacityReservationID)
			Expect(err).ToNot(HaveOccurred(), "Failed to cancel capacityreservation")
		})

		Eventually(func() error {
			return cl.Create(ctx, awsMachineTemplate)
		}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "Failed to create awsmachinetemplate")
		machineSetParams = framework.UpdateCAPIMachineSetName("aws-machineset-84243-only", machineSetParams)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")

		// Check for capacity issues or successful provisioning
		var hasCapacityIssues bool
		var capacityErrorMessage string
		Eventually(func() bool {
			machines, err := framework.GetCAPIMachinesFromMachineSet(ctx, cl, machineSet)
			if err != nil || len(machines) == 0 {
				return false
			}

			// Check if any machine is running (successful provisioning)
			for _, machine := range machines {
				if machine.Status.Phase == "Running" {
					return true
				}
			}

			// Check for capacity issues
			capacityErrorKeys := []string{"ReservationCapacityExceeded"}
			for _, machine := range machines {
				hasCapacityIssue, errorMessage, err := framework.HasCAPIInsufficientCapacity(ctx, cl, machine, capacityErrorKeys)
				if err == nil && hasCapacityIssue {
					hasCapacityIssues = true
					capacityErrorMessage = errorMessage
					By("Detected capacity issue - this is expected behavior when no capacity reservation is available")

					return true
				}
			}

			return false
		}, framework.WaitLong, framework.RetryMedium).Should(BeTrue(), "machine should either run successfully or encounter expected capacity issues")

		// Only continue with instance verification if no capacity issues
		if !hasCapacityIssues {
			framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)

			instance := getAWSInstanceConfig(ctx, cl, oc, machineSet)
			Expect(instance.CapacityReservationSpecification.CapacityReservationPreference).To(Equal(ptr.To("capacity-reservations-only")), fmt.Sprintf("Instance should have capacity reservation preference set to '%s'", "capacity-reservations-only"))
			Expect(instance.CapacityReservationId).To(Equal(ptr.To(capacityReservationID)), "Instance should be using the expected capacity reservation")
		} else {
			By(fmt.Sprintf("Test completed with expected capacity issue: %s", capacityErrorMessage))
		}
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

// createAWSCAPIMachineSetWithRetry creates a CAPI MachineSet with retry logic for capacity constraints.
// It tries different instance types when encountering insufficient capacity errors.
func createAWSCAPIMachineSetWithRetry(ctx context.Context, cl client.Client, machineSetName string, clusterName string, mapiDefaultProviderSpec *mapiv1.AWSMachineProviderConfig, maxRetries int, templateConfigurator func(*awsv1.AWSMachineTemplate, string)) (*clusterv1beta1.MachineSet, *awsv1.AWSMachineTemplate, bool) {
	machineSetReady := false

	// Get the current cluster architecture
	workers, err := framework.GetWorkerMachineSets(ctx, cl)
	Expect(err).ToNot(HaveOccurred(), "listing Worker MachineSets should not error.")
	Expect(len(workers)).To(BeNumerically(">=", 1), "expected at least one worker MachineSet to exist")

	arch, err := framework.GetArchitectureFromMachineSetNodes(ctx, cl, workers[0])
	Expect(err).NotTo(HaveOccurred(), "unable to get the architecture for the machine set")

	// Select alternative instance types based on architecture
	var alternativeInstanceTypes []string

	switch arch {
	case "arm64":
		alternativeInstanceTypes = []string{"m6g.large", "m6g.xlarge", "c6g.large", "c6g.xlarge"}
	default: // x86_64/amd64
		alternativeInstanceTypes = []string{"m6i.large", "m5.large", "m6i.xlarge", "m5.xlarge"}
	}

	var machineSet *clusterv1beta1.MachineSet

	var awsMachineTemplate *awsv1.AWSMachineTemplate

	for i, instanceType := range alternativeInstanceTypes {
		if i >= maxRetries {
			// If there are many alternatives, only try the specified number of times
			break
		}

		By(fmt.Sprintf("Attempting creation of CAPI MachineSet/AWSMachineTemplate with instance type %s for %s architecture", instanceType, arch))

		// Create AWS machine template
		awsMachineTemplate = newAWSMachineTemplate(awsMachineTemplateName, mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.InstanceType = instanceType

		// Apply specific configuration (spot, dedicated, etc.)
		templateConfigurator(awsMachineTemplate, instanceType)

		Eventually(func() error {
			return cl.Create(ctx, awsMachineTemplate)
		}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "Failed to create awsmachinetemplate")

		machineSetParams := framework.NewCAPIMachineSetParams(
			machineSetName,
			clusterName,
			mapiDefaultProviderSpec.Placement.AvailabilityZone,
			1,
			corev1.ObjectReference{
				Kind:       "AWSMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       awsMachineTemplateName,
			},
		)

		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")

		// Define AWS-specific capacity error keys
		awsCapacityErrorKeys := []string{"InsufficientInstanceCapacity"}
		err = framework.WaitForCAPIMachinesRunningWithRetry(ctx, cl, machineSet.Name, awsCapacityErrorKeys)

		if errors.Is(err, framework.ErrMachineNotProvisionedInsufficientCloudCapacity) {
			By("Cleaning up failed CAPI MachineSet/AWSMachineTemplate creation attempt because of failed provisioning due to insufficient capacity")
			// If machineSet cannot scale up due to insufficient capacity, try again with different instance type
			framework.DeleteCAPIMachineSets(ctx, cl, machineSet)
			framework.WaitForCAPIMachineSetsDeleted(ctx, cl, machineSet)
			framework.DeleteObjects(ctx, cl, awsMachineTemplate)

			continue
		}

		Expect(err).ToNot(HaveOccurred(), "Error while waiting for CAPI MachineSet Machines to be ready")

		machineSetReady = true

		break // MachineSet created successfully
	}

	return machineSet, awsMachineTemplate, machineSetReady
}

// getAWSInstanceConfig gets AWS instance configuration.
func getAWSInstanceConfig(ctx context.Context, cl client.Client, oc *gatherer.CLI, machineSet *clusterv1beta1.MachineSet) *ec2.Instance {
	By("Get AWS instance configuration")

	machines, err := framework.GetCAPIMachinesFromMachineSet(ctx, cl, machineSet)
	Expect(err).ToNot(HaveOccurred(), "Failed to get machines from machineset")
	Expect(machines).To(HaveLen(1), "Expected exactly one machine")

	machine := machines[0]
	infraMachine, err := framework.GetCAPIInfraMachine(ctx, cl, machine)
	Expect(err).ToNot(HaveOccurred(), "Failed to get InfraMachine")

	instanceID, found, err := unstructured.NestedString(infraMachine.Object, "spec", "instanceID")
	Expect(err).ToNot(HaveOccurred(), "Failed to get instanceID from InfraMachine")
	Expect(found).To(BeTrue(), "InfraMachine should have an instanceID")
	Expect(instanceID).ToNot(BeEmpty(), "instanceID should not be empty")

	awsClient := framework.NewAwsClient(framework.GetCredentialsFromCluster(oc))
	instance, err := awsClient.DescribeInstance(instanceID)
	Expect(err).ToNot(HaveOccurred(), "Failed to describe instaces")

	return instance
}
