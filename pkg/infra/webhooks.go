package infra

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	gcp "github.com/openshift/cluster-api-provider-gcp/pkg/apis/gcpprovider/v1beta1"
	mapiv1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	vsphere "github.com/openshift/machine-api-operator/pkg/apis/vsphereprovider/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	aws "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	azure "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:Machines] Webhooks", func() {
	var client runtimeclient.Client
	var platform configv1.PlatformType
	var machineSetParams framework.MachineSetParams

	var delObjects map[string]runtime.Object

	var ctx = context.Background()

	BeforeEach(func() {
		delObjects = make(map[string]runtime.Object)

		var err error
		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		// Only run on platforms that have webhooks
		clusterInfra, err := framework.GetInfrastructure(client)
		Expect(err).NotTo(HaveOccurred())
		platform = clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.AzurePlatformType, configv1.GCPPlatformType, configv1.VSpherePlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not have webhooks, skipping.", platform))
		}

		machineSetParams = framework.BuildMachineSetParams(client, 1)
		ps, err := createMinimalProviderSpec(platform, machineSetParams.ProviderSpec)
		Expect(err).ToNot(HaveOccurred())
		machineSetParams.ProviderSpec = ps
	})

	AfterEach(func() {
		Expect(deleteObjects(client, delObjects)).To(Succeed())
	})

	It("should be able to create a machine from a minimal providerSpec", func() {
		machine := &mapiv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-webhook-", machineSetParams.Name),
				Namespace:    framework.MachineAPINamespace,
				Labels:       machineSetParams.Labels,
			},
			Spec: mapiv1.MachineSpec{
				ProviderSpec: *machineSetParams.ProviderSpec,
			},
		}
		Expect(client.Create(ctx, machine)).To(Succeed())

		Eventually(func() error {
			m, err := framework.GetMachine(client, machine.Name)
			if err != nil {
				return err
			}
			running := framework.FilterRunningMachines([]*mapiv1.Machine{m})
			if len(running) == 0 {
				return fmt.Errorf("machine not yet running")
			}
			return nil
		}, framework.WaitLong, framework.RetryMedium).Should(Succeed())
	})

	It("should be able to create machines from a machineset with a minimal providerSpec", func() {
		machineSet, err := framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())

		framework.WaitForMachineSet(client, machineSet.Name)
	})
})

func createMinimalProviderSpec(platform configv1.PlatformType, ps *mapiv1.ProviderSpec) (*mapiv1.ProviderSpec, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return minimalAWSProviderSpec(ps)
	case configv1.AzurePlatformType:
		return minimalAzureProviderSpec(ps)
	case configv1.GCPPlatformType:
		return minimalGCPProviderSpec(ps)
	case configv1.VSpherePlatformType:
		return minimalVSphereProviderSpec(ps)
	default:
		// Should have skipped before this point
		return nil, fmt.Errorf("Unexpected platform: %s", platform)
	}
}

func minimalAWSProviderSpec(ps *mapiv1.ProviderSpec) (*mapiv1.ProviderSpec, error) {
	fullProviderSpec := &aws.AWSMachineProviderConfig{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &mapiv1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &aws.AWSMachineProviderConfig{
				AMI:       fullProviderSpec.AMI,
				Placement: fullProviderSpec.Placement,
			},
		},
	}, nil
}

func minimalAzureProviderSpec(ps *mapiv1.ProviderSpec) (*mapiv1.ProviderSpec, error) {
	fullProviderSpec := &azure.AzureMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &mapiv1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &azure.AzureMachineProviderSpec{
				Location: fullProviderSpec.Location,
				OSDisk: azure.OSDisk{
					DiskSizeGB: fullProviderSpec.OSDisk.DiskSizeGB,
				},
			},
		},
	}, nil
}

func minimalGCPProviderSpec(ps *mapiv1.ProviderSpec) (*mapiv1.ProviderSpec, error) {
	fullProviderSpec := &gcp.GCPMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &mapiv1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &gcp.GCPMachineProviderSpec{
				Region: fullProviderSpec.Region,
				Zone:   fullProviderSpec.Zone,
			},
		},
	}, nil
}

func minimalVSphereProviderSpec(ps *mapiv1.ProviderSpec) (*mapiv1.ProviderSpec, error) {
	fullProviderSpec := &vsphere.VSphereMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &mapiv1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &vsphere.VSphereMachineProviderSpec{
				Template: fullProviderSpec.Template,
				Workspace: &vsphere.Workspace{
					Datacenter: fullProviderSpec.Workspace.Datacenter,
					Server:     fullProviderSpec.Workspace.Server,
				},
				Network: vsphere.NetworkSpec{
					Devices: fullProviderSpec.Network.Devices,
				},
			},
		},
	}, nil
}
