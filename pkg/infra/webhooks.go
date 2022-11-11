package infra

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("[Feature:Machines] Webhooks", func() {
	var client runtimeclient.Client
	var platform configv1.PlatformType
	var machineSetParams framework.MachineSetParams
	var testSelector *metav1.LabelSelector

	var gatherer *gatherer.StateGatherer

	var ctx = context.Background()

	BeforeEach(func() {
		var err error
		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred())

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())

		// Only run on platforms that have webhooks
		clusterInfra, err := framework.GetInfrastructure(client)
		Expect(err).NotTo(HaveOccurred())
		platform = clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.AzurePlatformType, configv1.GCPPlatformType, configv1.VSpherePlatformType, configv1.PowerVSPlatformType, configv1.NutanixPlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not have webhooks, skipping.", platform))
		}

		machineSetParams = framework.BuildMachineSetParams(client, 1)
		ps, err := createMinimalProviderSpec(platform, machineSetParams.ProviderSpec)
		Expect(err).ToNot(HaveOccurred())
		machineSetParams.ProviderSpec = ps

		// All machines/machinesets created in this test should match these labels
		testSelector = &metav1.LabelSelector{
			MatchLabels: machineSetParams.Labels,
		}

		By("Checking the webhook configurations are synced", func() {
			Eventually(func() bool {
				return framework.IsMutatingWebhookConfigurationSynced(client)
			}, framework.WaitShort).Should(BeTrue(), "MutatingWebhookConfiguration must be synced before running these tests")

			Eventually(func() bool {
				return framework.IsValidatingWebhookConfigurationSynced(client)
			}, framework.WaitShort).Should(BeTrue(), "ValidingWebhookConfiguration must be synced before running these tests")
		})
	})

	AfterEach(func() {
		testDescription := CurrentGinkgoTestDescription()
		if testDescription.Failed == true {
			Expect(gatherer.WithTestDescription(testDescription).GatherAll()).To(Succeed())
		}

		machineSets, err := framework.GetMachineSets(client, testSelector)
		Expect(err).ToNot(HaveOccurred())
		framework.DeleteMachineSets(client, machineSets...)
		framework.WaitForMachineSetsDeleted(client, machineSets...)

		machines, err := framework.GetMachines(client, testSelector)
		Expect(err).ToNot(HaveOccurred())
		framework.DeleteMachines(client, machines...)
		framework.WaitForMachinesDeleted(client, machines...)
	})

	It("should be able to create a machine from a minimal providerSpec", func() {
		machine := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-webhook-", machineSetParams.Name),
				Namespace:    framework.MachineAPINamespace,
				Labels:       machineSetParams.Labels,
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: *machineSetParams.ProviderSpec,
			},
		}
		Expect(client.Create(ctx, machine)).To(Succeed())

		Eventually(func() error {
			m, err := framework.GetMachine(client, machine.Name)
			if err != nil {
				return err
			}
			running := framework.FilterRunningMachines([]*machinev1beta1.Machine{m})
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

	It("should return an error when removing required fields from the Machine providerSpec", func() {
		machine := &machinev1beta1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-webhook-", machineSetParams.Name),
				Namespace:    framework.MachineAPINamespace,
				Labels:       machineSetParams.Labels,
			},
			Spec: machinev1beta1.MachineSpec{
				ProviderSpec: *machineSetParams.ProviderSpec,
			},
		}
		Expect(client.Create(ctx, machine)).To(Succeed())

		updated := false
		for !updated {
			machine, err := framework.GetMachine(client, machine.Name)
			Expect(err).ToNot(HaveOccurred())

			minimalSpec, err := createMinimalProviderSpec(platform, &machine.Spec.ProviderSpec)
			Expect(err).ToNot(HaveOccurred())

			machine.Spec.ProviderSpec = *minimalSpec
			err = client.Update(ctx, machine)
			if apierrors.IsConflict(err) {
				// Try again if there was a conflict
				continue
			}
			// No conflict, so the update "worked"
			updated = true
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("admission webhook \"validation.machine.machine.openshift.io\" denied the request")))
		}
	})

	It("should return an error when removing required fields from the MachineSet providerSpec", func() {
		machineSet, err := framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())

		updated := false
		for !updated {
			machineSet, err = framework.GetMachineSet(client, machineSet.Name)
			Expect(err).ToNot(HaveOccurred())

			minimalSpec, err := createMinimalProviderSpec(platform, &machineSet.Spec.Template.Spec.ProviderSpec)
			Expect(err).ToNot(HaveOccurred())

			machineSet.Spec.Template.Spec.ProviderSpec = *minimalSpec
			err = client.Update(ctx, machineSet)
			if apierrors.IsConflict(err) {
				// Try again if there was a conflict
				continue
			}

			// No conflict, so the update "worked"
			updated = true
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("admission webhook \"validation.machineset.machine.openshift.io\" denied the request")))
		}

	})
})

func createMinimalProviderSpec(platform configv1.PlatformType, ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return minimalAWSProviderSpec(ps)
	case configv1.AzurePlatformType:
		return minimalAzureProviderSpec(ps)
	case configv1.GCPPlatformType:
		return minimalGCPProviderSpec(ps)
	case configv1.VSpherePlatformType:
		return minimalVSphereProviderSpec(ps)
	case configv1.PowerVSPlatformType:
		return minimalPowerVSProviderSpec(ps)
	case configv1.NutanixPlatformType:
		return minimalNutanixProviderSpec(ps)
	default:
		// Should have skipped before this point
		return nil, fmt.Errorf("Unexpected platform: %s", platform)
	}
}

func minimalAWSProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	fullProviderSpec := &machinev1beta1.AWSMachineProviderConfig{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &machinev1beta1.AWSMachineProviderConfig{
				AMI:                fullProviderSpec.AMI,
				Placement:          fullProviderSpec.Placement,
				Subnet:             *fullProviderSpec.Subnet.DeepCopy(),
				IAMInstanceProfile: fullProviderSpec.IAMInstanceProfile.DeepCopy(),
				SecurityGroups:     fullProviderSpec.SecurityGroups,
			},
		},
	}, nil
}

func minimalAzureProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	fullProviderSpec := &machinev1beta1.AzureMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &machinev1beta1.AzureMachineProviderSpec{
				Location: fullProviderSpec.Location,
				OSDisk: machinev1beta1.OSDisk{
					DiskSizeGB: fullProviderSpec.OSDisk.DiskSizeGB,
				},
			},
		},
	}, nil
}

func minimalGCPProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	fullProviderSpec := &machinev1beta1.GCPMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, fullProviderSpec)
	if err != nil {
		return nil, err
	}
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &machinev1beta1.GCPMachineProviderSpec{
				Region:          fullProviderSpec.Region,
				Zone:            fullProviderSpec.Zone,
				ServiceAccounts: fullProviderSpec.ServiceAccounts,
			},
		},
	}, nil
}

func minimalVSphereProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	providerSpec := &machinev1beta1.VSphereMachineProviderSpec{}
	err := json.Unmarshal(ps.Value.Raw, providerSpec)
	if err != nil {
		return nil, err
	}
	// For vSphere only these 2 fields are defaultable
	providerSpec.UserDataSecret = nil
	providerSpec.CredentialsSecret = nil
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: providerSpec,
		},
	}, nil
}

func minimalNutanixProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	providerSpec := &machinev1.NutanixMachineProviderConfig{}
	err := json.Unmarshal(ps.Value.Raw, providerSpec)
	if err != nil {
		return nil, err
	}

	// For nutanix only these 2 fields are defaultable
	providerSpec.UserDataSecret = nil
	providerSpec.CredentialsSecret = nil
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: providerSpec,
		},
	}, nil
}

func minimalPowerVSProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	providerSpec := &machinev1.PowerVSMachineProviderConfig{}
	err := json.Unmarshal(ps.Value.Raw, providerSpec)
	if err != nil {
		return nil, err
	}
	providerSpec.UserDataSecret = nil
	providerSpec.CredentialsSecret = nil
	providerSpec.SystemType = ""
	providerSpec.ProcessorType = ""
	providerSpec.MemoryGiB = 0
	providerSpec.Processors = intstr.FromString("")
	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: providerSpec,
		},
	}, nil
}
