package capi

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
	framework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	yaml "sigs.k8s.io/yaml"
)

const (
	infraAPIVersion            = "infrastructure.cluster.x-k8s.io/v1beta1"
	gcpMachineTemplateName     = "gcp-machine-template"
	OnHostMaintenanceTerminate = "Terminate"
	OnHostMaintenanceMigrate   = "Migrate"
)

var (
	clusterName string
	cl          client.Client
)

var _ = Describe("Cluster API GCP MachineSet", framework.LabelCAPI, framework.LabelDisruptive, Ordered, func() {
	var gcpMachineTemplate *gcpv1.GCPMachineTemplate
	var machineSet *clusterv1.MachineSet
	var mapiMachineSpec *mapiv1.GCPMachineProviderSpec
	var ctx context.Context
	var platform configv1.PlatformType
	var clusterName string
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
		oc, _ := framework.NewCLI()
		framework.SkipIfNotTechPreviewNoUpgrade(oc, cl)

		infra, err := framework.GetInfrastructure(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")
		Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "infrastructure name was empty on Infrastructure.Status.")
		clusterName = infra.Status.InfrastructureName

		framework.CreateCoreCluster(ctx, cl, clusterName, "GCPCluster")
		mapiMachineSpec = getGCPMAPIProviderSpec(cl)
	})
	AfterEach(func() {
		if platform != configv1.GCPPlatformType {
			// Because AfterEach always runs, even when tests are skipped, we have to
			// explicitly skip it here for other platforms.
			Skip("Skipping GCP E2E tests")
		}
		framework.DeleteCAPIMachineSets(ctx, cl, machineSet)
		framework.WaitForCAPIMachineSetsDeleted(ctx, cl, machineSet)
		framework.DeleteObjects(ctx, cl, gcpMachineTemplate)
	})
	// OCP-77825 capi created instances support disks pd-ssd and pd-standard
	// author: miyadav@redhat.com
	DescribeTable("should be able to run a machine with disk types",
		func(expectedDiskType gcpv1.DiskType) {
			mapiProviderSpec := getGCPMAPIProviderSpec(cl)
			Expect(mapiProviderSpec).ToNot(BeNil())
			gcpMachineTemplate = createGCPMachineTemplate(mapiProviderSpec)
			gcpMachineTemplate.Spec.Template.Spec.RootDeviceType = &expectedDiskType
			mapiProviderSpec.OnHostMaintenance = OnHostMaintenanceTerminate
			gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)

			if err := cl.Create(ctx, gcpMachineTemplate); err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}

			machineSet, _ = framework.CreateCAPIMachineSet(ctx, cl, framework.NewCAPIMachineSetParams(
				"gcp-machineset-77825",
				clusterName,
				mapiMachineSpec.Zone,
				1,
				corev1.ObjectReference{
					Kind:       "GCPMachineTemplate",
					APIVersion: infraAPIVersion,
					Name:       gcpMachineTemplateName,
				},
			))
			Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset")
			framework.WaitForCAPIMachinesRunning(framework.GetContext(), cl, machineSet.Name)
		},
		Entry("Disk type pd-standard", gcpv1.PdStandardDiskType),
		Entry("Disk type pd-ssd", gcpv1.PdSsdDiskType),
	)
	// OCP-74795 - add support for shielded VMs - It takes defaults if configs are not supported; eg-vtpm alone set to Enabled will result in IntergrityMonitoring also as Enabled
	// doesn't matter what we pass - all enabled/secureboot only disabled(default) are valid changes which we can apply
	// author: miyadav@redhat.com
	DescribeTable("should configure Shielded VM options correctly",
		func(enableSecureBoot gcpv1.SecureBootPolicy, enableVtpm gcpv1.VirtualizedTrustedPlatformModulePolicy, enableIntegrityMonitoring gcpv1.IntegrityMonitoringPolicy) {
			mapiProviderSpec := getGCPMAPIProviderSpec(cl)
			Expect(mapiProviderSpec).ToNot(BeNil())
			gcpMachineTemplate = createGCPMachineTemplate(mapiProviderSpec)
			mapiProviderSpec.OnHostMaintenance = OnHostMaintenanceMigrate
			gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)
			gcpMachineTemplate.Spec.Template.Spec.ShieldedInstanceConfig = &gcpv1.GCPShieldedInstanceConfig{
				SecureBoot:                       enableSecureBoot,
				VirtualizedTrustedPlatformModule: enableVtpm,
				IntegrityMonitoring:              enableIntegrityMonitoring,
			}
			if err := cl.Create(ctx, gcpMachineTemplate); err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}

			machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, framework.NewCAPIMachineSetParams(
				"gcp-machineset-shieldedvm-74795",
				clusterName,
				mapiMachineSpec.Zone,
				1,
				corev1.ObjectReference{
					Kind:       "GCPMachineTemplate",
					APIVersion: infraAPIVersion,
					Name:       gcpMachineTemplateName,
				},
			))
			Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI machineset with Shielded VM config")

			framework.WaitForCAPIMachinesRunning(framework.GetContext(), cl, machineSet.Name)

			By("Verifying the Shielded VM configuration on the created GCP MachineTemplate")
			createdTemplate := &gcpv1.GCPMachineTemplate{}
			Expect(cl.Get(ctx, client.ObjectKey{
				Namespace: framework.ClusterAPINamespace,
				Name:      gcpMachineTemplateName,
			}, createdTemplate)).To(Succeed())
			Expect(createdTemplate.Spec.Template.Spec.ShieldedInstanceConfig).ToNot(BeNil())
			Expect(fmt.Sprintf("%v", createdTemplate.Spec.Template.Spec.ShieldedInstanceConfig.SecureBoot)).To(Equal(fmt.Sprintf("%v", enableSecureBoot)))
			Expect(fmt.Sprintf("%v", createdTemplate.Spec.Template.Spec.ShieldedInstanceConfig.VirtualizedTrustedPlatformModule)).To(Equal(fmt.Sprintf("%v", enableVtpm)))
			Expect(fmt.Sprintf("%v", createdTemplate.Spec.Template.Spec.ShieldedInstanceConfig.IntegrityMonitoring)).To(Equal(fmt.Sprintf("%v", enableIntegrityMonitoring)))
		},
		Entry("all Shielded VM options enabled", gcpv1.SecureBootPolicyEnabled, gcpv1.VirtualizedTrustedPlatformModulePolicyEnabled, gcpv1.IntegrityMonitoringPolicyEnabled),
		Entry("only SecureBoot enabled", gcpv1.SecureBootPolicyEnabled, gcpv1.VirtualizedTrustedPlatformModulePolicyDisabled, gcpv1.IntegrityMonitoringPolicyDisabled),
		/*Below configs doesn't make difference due to defaulting conditions of shielded VMs
		Entry("only Vtpm enabled", gcpv1.SecureBootPolicyDisabled, gcpv1.VirtualizedTrustedPlatformModulePolicyEnabled, gcpv1.IntegrityMonitoringPolicyDisabled),
		Entry("only IntegrityMonitoring enabled", gcpv1.SecureBootPolicyDisabled, gcpv1.VirtualizedTrustedPlatformModulePolicyDisabled, gcpv1.IntegrityMonitoringPolicyEnabled),
		Entry("SecureBoot and Vtpm enabled", gcpv1.SecureBootPolicyEnabled, mapiv1.VirtualizedTrustedPlatformModulePolicyEnabled, gcpv1.IntegrityMonitoringPolicyDisabled),
		Entry("SecureBoot and IntegrityMonitoring enabled", gcpv1.SecureBootPolicyEnabled, gcpv1.VirtualizedTrustedPlatformModulePolicyDisabled, gcpv1.IntegrityMonitoringPolicyEnabled),
		Entry("all Shielded VM options disabled", gcpv1.SecureBootPolicyDisabled, gcpv1.VirtualizedTrustedPlatformModulePolicyDisabled, gcpv1.IntegrityMonitoringPolicyDisabled),
		*/
	)
	// OCP-74703 - Create confidential compute VMs on gcp
	// author: miyadav@redhat.com
	// Define constants as variable for this case to pass values properly
	confidentialComputeEnabled := gcpv1.ConfidentialComputePolicyEnabled
	confidentialComputeDisabled := gcpv1.ConfidentialComputePolicyDisabled
	DescribeTable("should configure Confidential VM correctly",
		func(confidentialCompute *gcpv1.ConfidentialComputePolicy) {
			mapiProviderSpec := getGCPMAPIProviderSpec(cl)
			Expect(mapiProviderSpec).ToNot(BeNil())
			gcpMachineTemplate = createGCPMachineTemplate(mapiProviderSpec)
			gcpMachineTemplate.Spec.Template.Spec.ConfidentialCompute = confidentialCompute
			gcpMachineTemplate.Spec.Template.Spec.InstanceType = "n2d-standard-4"

			if *confidentialCompute == gcpv1.ConfidentialComputePolicyEnabled {
				mapiProviderSpec.OnHostMaintenance = OnHostMaintenanceTerminate
				gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)
			} else {
				mapiProviderSpec.OnHostMaintenance = "Migrate"
				gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)
			}

			if err := cl.Create(ctx, gcpMachineTemplate); err != nil && !apierrors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}

			By("Creating a MachineSet for Confidential VM")
			machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, framework.NewCAPIMachineSetParams(
				"gcp-machineset-confidential-74703",
				clusterName,
				mapiProviderSpec.Zone,
				1,
				corev1.ObjectReference{
					Kind:       "GCPMachineTemplate",
					APIVersion: infraAPIVersion,
					Name:       gcpMachineTemplateName,
				},
			))
			Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI MachineSet with Confidential VM configuration")

			framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)

			By("Verifying the Confidential VM configuration on the created GCP MachineTemplate")
			createdTemplate := &gcpv1.GCPMachineTemplate{}
			Expect(cl.Get(framework.GetContext(), client.ObjectKey{
				Namespace: framework.ClusterAPINamespace,
				Name:      gcpMachineTemplateName,
			}, createdTemplate)).To(Succeed())
			var confidentialComputevalue = *createdTemplate.Spec.Template.Spec.ConfidentialCompute
			Expect(fmt.Sprintf("%v", confidentialComputevalue)).To(Equal(fmt.Sprintf("%v", *confidentialCompute)))
		},
		Entry("Confidential Compute enabled", &confidentialComputeEnabled),
		Entry("Confidential Compute disabled", &confidentialComputeDisabled),
	)
	// OCP-74732 GPU machine can be provisioned successfully by capi machineset
	// author: miyadav@redhat.com
	It("should provision GPU machine successfully", func() {
		mapiProviderSpec := getGCPMAPIProviderSpec(cl)
		Expect(mapiProviderSpec).ToNot(BeNil())
		gcpMachineTemplate = createGCPMachineTemplate(mapiProviderSpec)
		gcpMachineTemplate.Spec.Template.Spec.InstanceType = "a2-highgpu-1g"
		mapiProviderSpec.OnHostMaintenance = OnHostMaintenanceTerminate
		gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)

		if err := cl.Create(ctx, gcpMachineTemplate); err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		By("Creating a MachineSet for GPU machine")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, framework.NewCAPIMachineSetParams(
			"gcp-machineset-gpu-74732",
			clusterName,
			mapiProviderSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "GCPMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       gcpMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI MachineSet with GPU instanceType")

		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)

		By("Verifying the GPU machinetype configuration on the created GCP MachineTemplate")
		createdTemplate := &gcpv1.GCPMachineTemplate{}
		Expect(cl.Get(framework.GetContext(), client.ObjectKey{
			Namespace: framework.ClusterAPINamespace,
			Name:      gcpMachineTemplateName,
		}, createdTemplate)).To(Succeed())
		var machineType = createdTemplate.Spec.Template.Spec.InstanceType
		Expect(machineType).To(Equal("a2-highgpu-1g"))
	},
	)
	// OCP-75792 Preemptible machines can be provisioned successfully by capi machineset
	// author: miyadav@redhat.com
	It("should provision Preemptible machine successfully", func() {
		mapiProviderSpec := getGCPMAPIProviderSpec(cl)
		Expect(mapiProviderSpec).ToNot(BeNil())
		gcpMachineTemplate = createGCPMachineTemplate(mapiProviderSpec)
		gcpMachineTemplate.Spec.Template.Spec.Preemptible = true
		mapiProviderSpec.OnHostMaintenance = OnHostMaintenanceTerminate
		gcpMachineTemplate.Spec.Template.Spec.OnHostMaintenance = (*gcpv1.HostMaintenancePolicy)(&mapiProviderSpec.OnHostMaintenance)

		if err := cl.Create(ctx, gcpMachineTemplate); err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		By("Creating a MachineSet for preeemptible machine")
		machineSet, err = framework.CreateCAPIMachineSet(ctx, cl, framework.NewCAPIMachineSetParams(
			"gcp-machineset-preemptible-75792",
			clusterName,
			mapiProviderSpec.Zone,
			1,
			corev1.ObjectReference{
				Kind:       "GCPMachineTemplate",
				APIVersion: infraAPIVersion,
				Name:       gcpMachineTemplateName,
			},
		))
		Expect(err).ToNot(HaveOccurred(), "Failed to create CAPI MachineSet with preemptible instanceType")

		framework.WaitForCAPIMachinesRunning(ctx, cl, machineSet.Name)

		By("Verifying the preemptible machinetype configuration on the created GCP MachineTemplate")
		createdTemplate := &gcpv1.GCPMachineTemplate{}
		Expect(cl.Get(framework.GetContext(), client.ObjectKey{
			Namespace: framework.ClusterAPINamespace,
			Name:      gcpMachineTemplateName,
		}, createdTemplate)).To(Succeed())
		var preemptible = createdTemplate.Spec.Template.Spec.Preemptible
		Expect(preemptible).To(Equal(true))
	})

})

func getGCPMAPIProviderSpec(cl client.Client) *mapiv1.GCPMachineProviderSpec {
	machineSetList := &mapiv1.MachineSetList{}

	Eventually(func() error {
		return cl.List(framework.GetContext(), machineSetList, client.InNamespace(framework.MachineAPINamespace))
	}, framework.WaitShort, framework.RetryShort).Should(Succeed(), "it should be able to list the MAPI machinesets")
	Expect(machineSetList.Items).ToNot(HaveLen(0), "expected the MAPI machinesets to be present")

	machineSet := machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil())

	providerSpec := &mapiv1.GCPMachineProviderSpec{}
	Expect(yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)).To(Succeed())

	return providerSpec
}

func createGCPMachineTemplate(mapiProviderSpec *mapiv1.GCPMachineProviderSpec) *gcpv1.GCPMachineTemplate {
	By("Creating GCP machine template")

	Expect(mapiProviderSpec).ToNot(BeNil())
	Expect(mapiProviderSpec.Disks).ToNot(BeNil())
	Expect(len(mapiProviderSpec.Disks)).To(BeNumerically(">", 0))
	Expect(mapiProviderSpec.Disks[0].Type).ToNot(BeEmpty())
	Expect(mapiProviderSpec.MachineType).ToNot(BeEmpty())
	Expect(mapiProviderSpec.NetworkInterfaces).ToNot(BeNil())
	Expect(len(mapiProviderSpec.NetworkInterfaces)).To(BeNumerically(">", 0))
	Expect(mapiProviderSpec.NetworkInterfaces[0].Subnetwork).ToNot(BeEmpty())
	Expect(mapiProviderSpec.ServiceAccounts).ToNot(BeNil())
	Expect(mapiProviderSpec.ServiceAccounts[0].Email).ToNot(BeEmpty())
	Expect(mapiProviderSpec.ServiceAccounts[0].Scopes).ToNot(BeNil())
	Expect(len(mapiProviderSpec.ServiceAccounts)).To(BeNumerically(">", 0))
	Expect(mapiProviderSpec.Tags).ToNot(BeNil())
	Expect(len(mapiProviderSpec.Tags)).To(BeNumerically(">", 0))

	ipForwardingDisabled := gcpv1.IPForwardingDisabled

	gcpMachineSpec := gcpv1.GCPMachineSpec{
		RootDeviceSize: mapiProviderSpec.Disks[0].SizeGB,
		InstanceType:   mapiProviderSpec.MachineType,
		Image:          &mapiProviderSpec.Disks[0].Image,
		Subnet:         &mapiProviderSpec.NetworkInterfaces[0].Subnetwork,
		ServiceAccount: &gcpv1.ServiceAccount{
			Email:  mapiProviderSpec.ServiceAccounts[0].Email,
			Scopes: mapiProviderSpec.ServiceAccounts[0].Scopes,
		},

		AdditionalNetworkTags: mapiProviderSpec.Tags,
		AdditionalLabels:      gcpv1.Labels{fmt.Sprintf("kubernetes-io-cluster-%s", clusterName): "owned"},
		IPForwarding:          &ipForwardingDisabled,
	}

	gcpMachineTemplate := &gcpv1.GCPMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gcpMachineTemplateName,
			Namespace: framework.ClusterAPINamespace,
		},
		Spec: gcpv1.GCPMachineTemplateSpec{
			Template: gcpv1.GCPMachineTemplateResource{
				Spec: gcpMachineSpec,
			},
		},
	}

	return gcpMachineTemplate
}
