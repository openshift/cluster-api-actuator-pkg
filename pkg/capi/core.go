package capi

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	gotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Cluster API MachineSet", framework.LabelCAPI, framework.LabelDisruptive, Ordered, func() {
	var awsMachineTemplate *awsv1.AWSMachineTemplate
	var azureMachineTemplate *azurev1.AzureMachineTemplate
	var gcpMachineTemplate *gcpv1.GCPMachineTemplate
	var awsMapiMachineSpec *mapiv1.AWSMachineProviderConfig
	var azureMapiMachineSpec *mapiv1.AzureMachineProviderSpec
	var gcpMapiMachineSpec *mapiv1.GCPMachineProviderSpec
	var machineSet *clusterv1beta1.MachineSet
	var client runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var clusterName string
	var failureDomain string
	var machineTemplateName string
	var kind string
	var machineSetParams framework.CAPIMachineSetParams
	var err error

	BeforeAll(func() {
		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(client)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, client)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")
		oc, _ := framework.NewCLI()
		framework.SkipIfNotTechPreviewNoUpgrade(oc, client)

		infra, err := framework.GetInfrastructure(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		clusterName = infra.Status.InfrastructureName

		switch platform {
		case configv1.AWSPlatformType:
			awsMapiMachineSpec = getDefaultAWSMAPIProviderSpec(client)
			failureDomain = awsMapiMachineSpec.Placement.AvailabilityZone
			kind = "AWSMachineTemplate"
		case configv1.AzurePlatformType:
			azureMapiMachineSpec = getAzureMAPIProviderSpec(client)
			failureDomain = azureMapiMachineSpec.Zone
			kind = "AzureMachineTemplate"
		case configv1.GCPPlatformType:
			gcpMapiMachineSpec = getGCPMAPIProviderSpec(client)
			failureDomain = gcpMapiMachineSpec.Zone
			kind = "GCPMachineTemplate"
		default:
			Skip(fmt.Sprintf("Platform %v does not support , skipping.", platform))
		}

	})

	AfterEach(func() {
		// if the current testing are skipped, we skip clean resources
		if CurrentSpecReport().State == gotypes.SpecStateSkipped {
			return
		}

		framework.DeleteCAPIMachineSets(ctx, client, machineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, client, machineSet)
		switch platform {
		case configv1.AWSPlatformType:
			framework.DeleteObjects(ctx, client, awsMachineTemplate)
		case configv1.AzurePlatformType:
			framework.DeleteObjects(ctx, client, azureMachineTemplate)
		case configv1.GCPPlatformType:
			framework.DeleteObjects(ctx, client, gcpMachineTemplate)
		default:
			Skip(fmt.Sprintf("Platform %v does not support , skipping.", platform))
		}
	})

	// OCP-75779 - [CAPI] Labels and annotations specified in a machineset should propagate to nodes.
	// author: huliu@redhat.com
	It("should be able to run a machine with labels and annotations and they are propagated to nodes", func() {
		switch platform {
		case configv1.AWSPlatformType:
			machineTemplateName = "awsmachinetemplate-75779"
			awsMachineTemplate = newAWSMachineTemplate(machineTemplateName, awsMapiMachineSpec)
			Expect(client.Create(ctx, awsMachineTemplate)).To(Succeed(), "Failed to create awsmachinetemplate")
		case configv1.AzurePlatformType:
			machineTemplateName = "azuremachinetemplate-75779"
			azureMachineTemplate = newAzureMachineTemplate(client, machineTemplateName, azureMapiMachineSpec)
			Expect(client.Create(ctx, azureMachineTemplate)).To(Succeed(), "Failed to create azuremachinetemplate")
		case configv1.GCPPlatformType:
			gcpMachineTemplate = createGCPMachineTemplate(gcpMapiMachineSpec)
			Expect(client.Create(ctx, gcpMachineTemplate)).To(Succeed(), "Failed to create gcpmachinetemplate")
			machineTemplateName = gcpMachineTemplate.Name
		default:
			Skip(fmt.Sprintf("Platform %v does not support , skipping.", platform))
		}

		machineSetParams = framework.NewCAPIMachineSetParams(
			"machineset-75779",
			clusterName,
			failureDomain,
			0,
			corev1.ObjectReference{
				Kind:       kind,
				APIVersion: infraAPIVersion,
				Name:       machineTemplateName,
			},
		)
		machineSet, err = framework.CreateCAPIMachineSet(ctx, client, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")

		machineSetCopy := machineSet.DeepCopy()
		machineSetCopy.Spec.Replicas = ptr.To(int32(1))
		machineSetCopy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		machineSetCopy.Spec.Template.ObjectMeta.Annotations["anno1key"] = "anno1value"
		machineSetCopy.Spec.Template.ObjectMeta.Labels["label1key"] = "label1value"

		err = client.Patch(ctx, machineSetCopy, runtimeclient.MergeFrom(machineSet))
		Expect(err).NotTo(HaveOccurred(), "Failed to patch "+machineSet.Name)
		framework.WaitForCAPIMachinesRunning(ctx, client, machineSet.Name)

		machines, err := framework.GetCAPIMachinesFromMachineSet(ctx, client, machineSet)
		Expect(err).NotTo(HaveOccurred(), "Failed to get machine from machineset")
		Expect(machines[0].ObjectMeta.Annotations["anno1key"]).Should(Equal("anno1value"))
		Expect(machines[0].ObjectMeta.Labels["label1key"]).Should(Equal("label1value"))

		node, err := framework.GetCAPINodeForMachine(ctx, client, machines[0])
		Expect(err).NotTo(HaveOccurred(), "Failed to get node from machine")
		Expect(node.ObjectMeta.Annotations["anno1key"]).Should(Equal("anno1value"))
		Expect(node.ObjectMeta.Labels["label1key"]).Should(Equal("label1value"))
	})
})
