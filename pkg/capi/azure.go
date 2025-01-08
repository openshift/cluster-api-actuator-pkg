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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ptr "k8s.io/utils/ptr"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	yaml "sigs.k8s.io/yaml"
)

const (
	azureMachineTemplateName        = "azure-machine-template"
	clusterSecretName               = "capz-manager-cluster-credential"
	capzManagerBootstrapCredentials = "capz-manager-bootstrap-credentials"
)

var _ = Describe("Cluster API Azure MachineSet", framework.LabelCAPI, framework.LabelDisruptive, Ordered, func() {
	var azureMachineTemplate *azurev1.AzureMachineTemplate
	var machineSet *clusterv1.MachineSet
	var mapiMachineSpec *mapiv1.AzureMachineProviderSpec
	var client runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var clusterName string
	var err error

	BeforeAll(func() {
		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(client)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, client)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")
		if platform != configv1.AzurePlatformType {
			Skip("Skipping Azure E2E tests")
		}
		oc, _ := framework.NewCLI()
		framework.SkipIfNotTechPreviewNoUpgrade(oc, client)

		infra, err := framework.GetInfrastructure(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		clusterName = infra.Status.InfrastructureName
		framework.CreateCoreCluster(ctx, client, clusterName, "AzureCluster")
		mapiMachineSpec = getAzureMAPIProviderSpec(client)
	})

	AfterEach(func() {
		// if the current testing are skipped, we skip clean resources
		if CurrentSpecReport().State == gotypes.SpecStateSkipped {
			return
		}

		framework.DeleteCAPIMachineSets(ctx, client, machineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, client, machineSet)
		framework.DeleteObjects(ctx, client, azureMachineTemplate)
	})

	// OCP-75884 - [CAPI] Create machineset with capi on Azure.
	// author: zhsun@redhat.com
	It("should be able to run a machine", func() {
		azureMachineTemplate = newAzureMachineTemplate(client, mapiMachineSpec)
		Expect(client.Create(ctx, azureMachineTemplate)).To(Succeed(), "Failed to create azuremachinetemplate")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, client, framework.NewCAPIMachineSetParams(
			"azure-machineset-75884",
			clusterName,
			mapiMachineSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "AzureMachineTemplate",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Name:       azureMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
		framework.WaitForCAPIMachinesRunning(framework.GetContext(), client, machineSet.Name)
	})

	// OCP-75959 - [CAPI] host-based disk encryption at VM on Azure platform.
	// author: zhsun@redhat.com
	// EncryptionAtHost feature is not enabled for dev subscription, added framework.LabelQEOnly
	It("should be able to run a machine with host-based disk encryption", framework.LabelQEOnly, func() {
		azureMachineTemplate = newAzureMachineTemplate(client, mapiMachineSpec)
		azureMachineTemplate.Spec.Template.Spec.SecurityProfile = &azurev1.SecurityProfile{
			EncryptionAtHost: ptr.To(true),
		}
		Expect(client.Create(ctx, azureMachineTemplate)).To(Succeed(), "Failed to create azuremachinetemplate")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, client, framework.NewCAPIMachineSetParams(
			"azure-machineset-75959",
			clusterName,
			mapiMachineSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "AzureMachineTemplate",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Name:       azureMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI host-based disk encryption machineset")
		framework.WaitForCAPIMachinesRunning(framework.GetContext(), client, machineSet.Name)

		By("Verifying the host-based disk encryption configuration on the created Azure MachineTemplate")
		Expect(azureMachineTemplate.Spec.Template.Spec.SecurityProfile.EncryptionAtHost).To(Equal(ptr.To(true)))
	})

	// OCP-75961 - [CAPI] Enable accelerated network via MachineSets on Azure.
	// author: zhsun@redhat.com
	It("should be able to run a machine with accelerated network", func() {
		azureMachineTemplate = newAzureMachineTemplate(client, mapiMachineSpec)
		azureMachineTemplate.Spec.Template.Spec.NetworkInterfaces = []azurev1.NetworkInterface{
			{
				AcceleratedNetworking: ptr.To(true),
				SubnetName:            mapiMachineSpec.Subnet,
			},
		}
		Expect(client.Create(ctx, azureMachineTemplate)).To(Succeed(), "Failed to create azuremachinetemplate")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, client, framework.NewCAPIMachineSetParams(
			"azure-machineset-75961",
			clusterName,
			mapiMachineSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "AzureMachineTemplate",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Name:       azureMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI accelerated network machineset")
		framework.WaitForCAPIMachinesRunning(framework.GetContext(), client, machineSet.Name)

		By("Verifying the accelerated network configuration on the created Azure MachineTemplate")
		Expect(azureMachineTemplate.Spec.Template.Spec.NetworkInterfaces[0].AcceleratedNetworking).To(Equal(ptr.To(true)))
	})

	// OCP-75972 - [CAPI] Spot instance can be created successfully with capi on azure.
	// author: zhsun@redhat.com
	It("should be able to run a machine with SpotVMOptions", func() {
		region := mapiMachineSpec.Location
		if region == "northcentralus" || region == "westus" || region == "usgovtexas" {
			Skip("Skipping this test scenario on the " + region + " region, because this region doesn't have zones")
		}
		azureMachineTemplate = newAzureMachineTemplate(client, mapiMachineSpec)
		azureMachineTemplate.Spec.Template.Spec.SpotVMOptions = &azurev1.SpotVMOptions{}
		Expect(client.Create(ctx, azureMachineTemplate)).To(Succeed(), "Failed to create azuremachinetemplate")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, client, framework.NewCAPIMachineSetParams(
			"azure-machineset-75972",
			clusterName,
			mapiMachineSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "AzureMachineTemplate",
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Name:       azureMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI spot machineset")
		framework.WaitForCAPIMachinesRunning(framework.GetContext(), client, machineSet.Name)
	})
})

func getAzureMAPIProviderSpec(client runtimeclient.Client) *mapiv1.AzureMachineProviderSpec {
	machineSetList := &mapiv1.MachineSetList{}

	Eventually(func() error {
		return client.List(framework.GetContext(), machineSetList, runtimeclient.InNamespace(framework.MachineAPINamespace))
	}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "it should be able to list the MAPI machinesets")
	Expect(machineSetList.Items).ToNot(HaveLen(0), "expected the MAPI machinesets to be present")

	machineSet := machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil(), "expected the MAPI machinesets ProviderSpec value to not be nil")

	providerSpec := &mapiv1.AzureMachineProviderSpec{}
	Expect(yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)).To(Succeed(), "it should be able to unmarshal the raw yaml into providerSpec")

	return providerSpec
}

func newAzureMachineTemplate(client runtimeclient.Client, mapiProviderSpec *mapiv1.AzureMachineProviderSpec) *azurev1.AzureMachineTemplate {
	By("Creating Azure machine template")
	Expect(mapiProviderSpec).ToNot(BeNil(), "expected the mapi ProviderSpec to not be nil")
	Expect(mapiProviderSpec.Subnet).ToNot(BeEmpty(), "expected the mapi Subnet to not be empty")
	Expect(mapiProviderSpec.AcceleratedNetworking).ToNot(BeNil(), "expected the mapi AcceleratedNetworking to not be nil")
	Expect(mapiProviderSpec.Image.ResourceID).ToNot(BeEmpty(), "expected the mapi ResourceID to not be empty")
	Expect(mapiProviderSpec.OSDisk.ManagedDisk.StorageAccountType).ToNot(BeEmpty(), "expected the mapi StorageAccountType to not be empty")
	Expect(mapiProviderSpec.OSDisk.DiskSizeGB).To(BeNumerically(">", 0), "expected the mapi DiskSizeGB > 0")
	Expect(mapiProviderSpec.OSDisk.OSType).ToNot(BeEmpty(), "expected the mapi OSType to not be empty")
	Expect(mapiProviderSpec.VMSize).ToNot(BeEmpty(), "expected the mapi VMSize to not be empty")

	azureCredentialsSecret := corev1.Secret{}
	azureCredentialsSecretKey := types.NamespacedName{Name: "capz-manager-bootstrap-credentials", Namespace: "openshift-cluster-api"}
	err := client.Get(context.Background(), azureCredentialsSecretKey, &azureCredentialsSecret)
	Expect(err).To(BeNil(), "capz-manager-bootstrap-credentials secret should exist")

	subscriptionID := azureCredentialsSecret.Data["azure_subscription_id"]
	azureImageID := fmt.Sprintf("/subscriptions/%s%s", subscriptionID, mapiProviderSpec.Image.ResourceID)
	azureMachineSpec := azurev1.AzureMachineSpec{
		Identity: azurev1.VMIdentityUserAssigned,
		UserAssignedIdentities: []azurev1.UserAssignedIdentity{
			{
				ProviderID: fmt.Sprintf("azure:///subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", subscriptionID, mapiProviderSpec.ResourceGroup, mapiProviderSpec.ManagedIdentity),
			},
		},
		NetworkInterfaces: []azurev1.NetworkInterface{
			{
				PrivateIPConfigs:      1,
				SubnetName:            mapiProviderSpec.Subnet,
				AcceleratedNetworking: &mapiProviderSpec.AcceleratedNetworking,
			},
		},
		Image: &azurev1.Image{
			ID: &azureImageID,
		},
		OSDisk: azurev1.OSDisk{
			DiskSizeGB: &mapiProviderSpec.OSDisk.DiskSizeGB,
			ManagedDisk: &azurev1.ManagedDiskParameters{
				StorageAccountType: mapiProviderSpec.OSDisk.ManagedDisk.StorageAccountType,
			},
			CachingType: mapiProviderSpec.OSDisk.CachingType,
			OSType:      mapiProviderSpec.OSDisk.OSType,
		},
		DisableExtensionOperations: ptr.To(true),
		SSHPublicKey:               mapiProviderSpec.SSHPublicKey,
		VMSize:                     mapiProviderSpec.VMSize,
	}

	azureMachineTemplate := &azurev1.AzureMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      azureMachineTemplateName,
			Namespace: framework.ClusterAPINamespace,
		},
		Spec: azurev1.AzureMachineTemplateSpec{
			Template: azurev1.AzureMachineTemplateResource{
				Spec: azureMachineSpec,
			},
		},
	}

	return azureMachineTemplate
}
