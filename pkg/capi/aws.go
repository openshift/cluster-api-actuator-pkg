package capi

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	capi "github.com/openshift/cluster-api-actuator-pkg/pkg/framework/capi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

const (
	awsMachineTemplateName = "aws-machine-template"
)

var (
	cl          client.Client
	ctx         = context.Background()
	platform    configv1.PlatformType
	clusterName string
)

var _ = Describe("Cluster API AWS MachineSet", Ordered, func() {
	var (
		awsMachineTemplate      *awsv1.AWSMachineTemplate
		machineSetParams        capi.MachineSetParams
		machineSet              *clusterv1.MachineSet
		mapiDefaultProviderSpec *mapiv1.AWSMachineProviderConfig
	)

	BeforeAll(func() {
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred())

		cl, err = client.New(cfg, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		infra := &configv1.Infrastructure{}
		infraName := client.ObjectKey{
			Name: infrastructureName,
		}
		Expect(cl.Get(ctx, infraName, infra)).To(Succeed())
		Expect(infra.Status.PlatformStatus).ToNot(BeNil())
		clusterName = infra.Status.InfrastructureName
		platform = infra.Status.PlatformStatus.Type

		machineSetParams = capi.NewMachineSetParams(
			"aws-machineset",
			clusterName,
			"",
			1,
			corev1.ObjectReference{
				Kind:       "AWSMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       awsMachineTemplateName,
			},
		)

		if platform != configv1.AWSPlatformType {
			Skip("Skipping AWS E2E tests")
		}
		oc, _ := framework.NewCLI()
		capi.SkipForCAPINotExist(oc)
		_, mapiDefaultProviderSpec = getDefaultAWSMAPIProviderSpec(cl)
		capi.CreateCoreCluster(cl, clusterName, "AWSCluster")
	})

	AfterEach(func() {
		if platform != configv1.AWSPlatformType {
			// Because AfterEach always runs, even when tests are skipped, we have to
			// explicitly skip it here for other platforms.
			Skip("Skipping AWS E2E tests")
		}
		capi.DeleteMachineSets(cl, machineSet)
		capi.WaitForMachineSetsDeleted(cl, machineSet)
		capi.DeleteObjects(cl, awsMachineTemplate)
	})

	It("should be able to run a machine with a default provider spec", func() {
		awsMachineTemplate = newAWSMachineTemplate(mapiDefaultProviderSpec)
		if err := cl.Create(ctx, awsMachineTemplate); err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		machineSet = capi.CreateMachineSet(cl, machineSetParams)
		capi.WaitForMachineSet(cl, machineSet.Name)
	})

	It("should be able to run a machine with cluster placement group", func() {
		awsMachineTemplate = newAWSMachineTemplate(mapiDefaultProviderSpec)
		awsMachineTemplate.Spec.Template.Spec.PlacementGroupName = "pgcluster"
		if err := cl.Create(ctx, awsMachineTemplate); err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		machineSet = capi.CreateMachineSet(cl, machineSetParams)
		capi.WaitForMachineSet(cl, machineSet.Name)
	})
})

func getDefaultAWSMAPIProviderSpec(cl client.Client) (*mapiv1.MachineSet, *mapiv1.AWSMachineProviderConfig) {
	machineSetList := &mapiv1.MachineSetList{}
	Expect(cl.List(ctx, machineSetList, client.InNamespace(capi.MAPINamespace))).To(Succeed())

	Expect(machineSetList.Items).ToNot(HaveLen(0))
	machineSet := &machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil())

	providerSpec := &mapiv1.AWSMachineProviderConfig{}
	Expect(yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)).To(Succeed())

	return machineSet, providerSpec
}

func newAWSMachineTemplate(mapiProviderSpec *mapiv1.AWSMachineProviderConfig) *awsv1.AWSMachineTemplate {
	By("Creating AWS machine template")

	Expect(mapiProviderSpec).ToNot(BeNil())
	Expect(mapiProviderSpec.IAMInstanceProfile).ToNot(BeNil())
	Expect(mapiProviderSpec.IAMInstanceProfile.ID).ToNot(BeNil())
	Expect(mapiProviderSpec.InstanceType).ToNot(BeEmpty())
	Expect(mapiProviderSpec.Placement.AvailabilityZone).ToNot(BeEmpty())
	Expect(mapiProviderSpec.AMI.ID).ToNot(BeNil())
	Expect(mapiProviderSpec.Subnet.Filters).ToNot(HaveLen(0))
	Expect(mapiProviderSpec.Subnet.Filters[0].Values).ToNot(HaveLen(0))
	Expect(mapiProviderSpec.SecurityGroups).ToNot(HaveLen(0))
	Expect(mapiProviderSpec.SecurityGroups[0].Filters).ToNot(HaveLen(0))
	Expect(mapiProviderSpec.SecurityGroups[0].Filters[0].Values).ToNot(HaveLen(0))

	uncompressedUserData := true

	awsMachineSpec := awsv1.AWSMachineSpec{
		UncompressedUserData: &uncompressedUserData,
		IAMInstanceProfile:   *mapiProviderSpec.IAMInstanceProfile.ID,
		InstanceType:         mapiProviderSpec.InstanceType,
		AMI: awsv1.AMIReference{
			ID: mapiProviderSpec.AMI.ID,
		},
		Ignition: &awsv1.Ignition{
			Version:     "3.4",
			StorageType: awsv1.IgnitionStorageTypeOptionUnencryptedUserData,
		},
		Subnet: &awsv1.AWSResourceReference{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: mapiProviderSpec.Subnet.Filters[0].Values,
				},
			},
		},
		AdditionalSecurityGroups: []awsv1.AWSResourceReference{
			{
				Filters: []awsv1.Filter{
					{
						Name:   "tag:Name",
						Values: mapiProviderSpec.SecurityGroups[0].Filters[0].Values,
					},
				},
			},
		},
	}

	awsMachineTemplate := &awsv1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      awsMachineTemplateName,
			Namespace: capi.CAPINamespace,
		},
		Spec: awsv1.AWSMachineTemplateSpec{
			Template: awsv1.AWSMachineTemplateResource{
				Spec: awsMachineSpec,
			},
		},
	}

	return awsMachineTemplate
}
