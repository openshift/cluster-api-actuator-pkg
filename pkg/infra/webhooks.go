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
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("Webhooks", framework.LabelMAPI, framework.LabelDisruptive, func() {
	var client runtimeclient.Client
	var platform configv1.PlatformType
	var machineSetParams framework.MachineSetParams
	var testSelector *metav1.LabelSelector

	var gatherer *gatherer.StateGatherer

	var ctx = context.Background()

	BeforeEach(func() {
		var err error
		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "StateGatherer should be able to be created")

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Controller-runtime client should be able to be created")

		// Only run on platforms that have webhooks
		clusterInfra, err := framework.GetInfrastructure(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Should be able to get Infrastructure")
		platform = clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.AzurePlatformType, configv1.GCPPlatformType, configv1.VSpherePlatformType, configv1.PowerVSPlatformType, configv1.NutanixPlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not have webhooks, skipping.", platform))
		}

		machineSetParams = framework.BuildMachineSetParams(ctx, client, 1)
		ps, err := createMinimalProviderSpec(platform, machineSetParams.ProviderSpec)
		Expect(err).ToNot(HaveOccurred(), "Should be able to generate MachineSet ProviderSpec")
		machineSetParams.ProviderSpec = ps

		// All machines/machinesets created in this test should match these labels
		testSelector = &metav1.LabelSelector{
			MatchLabels: machineSetParams.Labels,
		}

		By("Checking the webhook configurations are synced", func() {
			Eventually(func() bool {
				return framework.IsMutatingWebhookConfigurationSynced(ctx, client)
			}, framework.WaitShort).Should(BeTrue(), "MutatingWebhookConfiguration must be synced before running these tests")

			Eventually(func() bool {
				return framework.IsValidatingWebhookConfigurationSynced(ctx, client)
			}, framework.WaitShort).Should(BeTrue(), "ValidingWebhookConfiguration must be synced before running these tests")
		})
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "StateGatherer should be able to gather resources")
		}

		machineSets, err := framework.GetMachineSets(client, testSelector)
		Expect(err).ToNot(HaveOccurred(), "Should be able to list test MachineSets")
		Expect(framework.DeleteMachineSets(client, machineSets...)).To(Succeed(), "Should be able to delete test MachineSets")
		framework.WaitForMachineSetsDeleted(ctx, client, machineSets...)

		machines, err := framework.GetMachines(ctx, client, testSelector)
		Expect(err).ToNot(HaveOccurred(), "Should be able to get test Machines")
		Expect(framework.DeleteMachines(ctx, client, machines...)).To(Succeed(), "Should be able to delete test Machines")
		framework.WaitForMachinesDeleted(client, machines...)
	})

	// Machines required for test: 1
	// Reason: It needs to verify that machine with minimal provider spec is able to go into running phase.
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
		Expect(client.Create(ctx, machine)).To(Succeed(), "Should be able to create Machine")

		Eventually(func() error {
			m, err := framework.GetMachine(client, machine.Name)
			if err != nil {
				return err
			}

			failed := framework.FilterMachines([]*machinev1beta1.Machine{m}, framework.MachinePhaseFailed)
			if len(failed) > 0 {
				reason := "failureReason not present in Machine.status"
				if m.Status.ErrorReason != nil {
					reason = string(*m.Status.ErrorReason)
				}
				message := "failureMessage not present in Machine.status"
				if m.Status.ErrorMessage != nil {
					message = *m.Status.ErrorMessage
				}
				klog.Errorf("Failed machine: %s, Reason: %s, Message: %s", m.Name, reason, message)
			}
			Expect(len(failed)).To(Equal(0), "zero machines should be in a Failed phase")

			running := framework.FilterRunningMachines([]*machinev1beta1.Machine{m})
			if len(running) == 0 {
				return fmt.Errorf("machine not yet running")
			}

			return nil
		}, framework.WaitLong, framework.RetryMedium).Should(Succeed(), "Machine should go into Running state")
	})

	// Machines required for test: 1
	// Reason: It needs to verify that machine created from the machineSet with minimal provider spec is able to go into running phase.
	It("should be able to create machines from a machineset with a minimal providerSpec", func() {
		machineSet, err := framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Should be able to create MachineSet")

		framework.WaitForMachineSet(ctx, client, machineSet.Name)
	})

	// Machines required for test: 1
	// Reason: We need a machine to test updating its providerSpec. We don't wait for this machine to be running.
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
		Expect(client.Create(ctx, machine)).To(Succeed(), "Should be able to create Machine")

		updated := false
		for !updated {
			machine, err := framework.GetMachine(client, machine.Name)
			Expect(err).ToNot(HaveOccurred(), "Should be able to get Machine")

			minimalSpec, err := createMinimalProviderSpec(platform, &machine.Spec.ProviderSpec)
			Expect(err).ToNot(HaveOccurred(), "Should be able to generate Machine's ProviderSpec")

			machine.Spec.ProviderSpec = *minimalSpec
			err = client.Update(ctx, machine)
			if apierrors.IsConflict(err) {
				// Try again if there was a conflict
				continue
			}

			// No conflict, so the update "worked"
			updated = true
			Expect(err).To(HaveOccurred(), "Should be able to update Machine")
			Expect(err).To(MatchError(ContainSubstring("admission webhook \"validation.machine.machine.openshift.io\" denied the request")), "Should get an admission webhook denied error back")
		}
	})

	// Machines required for test: 0
	// Reason: We don't need to start creating the machine, because we are only testing the machineSet webhook.
	It("should return an error when removing required fields from the MachineSet providerSpec", func() {
		machineSetParams.Replicas = 0
		machineSet, err := framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "Should be able to create MachineSet")

		updated := false
		for !updated {
			machineSet, err = framework.GetMachineSet(ctx, client, machineSet.Name)
			Expect(err).ToNot(HaveOccurred(), "Should be able to get MachineSet")

			minimalSpec, err := createMinimalProviderSpec(platform, &machineSet.Spec.Template.Spec.ProviderSpec)
			Expect(err).ToNot(HaveOccurred(), "Should be able to generate Machine's ProviderSpec")

			machineSet.Spec.Template.Spec.ProviderSpec = *minimalSpec
			err = client.Update(ctx, machineSet)
			if apierrors.IsConflict(err) {
				// Try again if there was a conflict
				continue
			}

			// No conflict, so the update "worked"
			updated = true
			Expect(err).To(HaveOccurred(), "Should be able to update MachineSet")
			Expect(err).To(MatchError(ContainSubstring("admission webhook \"validation.machineset.machine.openshift.io\" denied the request")), "Should get an admission webhook denied error back")
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
		return nil, fmt.Errorf("unexpected platform: %s", platform)
	}
}

func minimalAWSProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	fullProviderSpec := &machinev1beta1.AWSMachineProviderConfig{}

	if err := json.Unmarshal(ps.Value.Raw, fullProviderSpec); err != nil {
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

	if err := json.Unmarshal(ps.Value.Raw, fullProviderSpec); err != nil {
		return nil, err
	}

	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &machinev1beta1.AzureMachineProviderSpec{
				Location: fullProviderSpec.Location,
				OSDisk: machinev1beta1.OSDisk{
					DiskSizeGB: fullProviderSpec.OSDisk.DiskSizeGB,
				},
				Vnet:                 fullProviderSpec.Vnet,
				Subnet:               fullProviderSpec.Subnet,
				NetworkResourceGroup: fullProviderSpec.NetworkResourceGroup,
			},
		},
	}, nil
}

func minimalGCPProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	fullProviderSpec := &machinev1beta1.GCPMachineProviderSpec{}

	if err := json.Unmarshal(ps.Value.Raw, fullProviderSpec); err != nil {
		return nil, err
	}

	return &machinev1beta1.ProviderSpec{
		Value: &runtime.RawExtension{
			Object: &machinev1beta1.GCPMachineProviderSpec{
				Region:          fullProviderSpec.Region,
				Zone:            fullProviderSpec.Zone,
				ServiceAccounts: fullProviderSpec.ServiceAccounts,
				NetworkInterfaces: []*machinev1beta1.GCPNetworkInterface{{
					Network:    fullProviderSpec.NetworkInterfaces[0].Network,
					Subnetwork: fullProviderSpec.NetworkInterfaces[0].Subnetwork,
					ProjectID:  fullProviderSpec.NetworkInterfaces[0].ProjectID,
				}},
			},
		},
	}, nil
}

func minimalVSphereProviderSpec(ps *machinev1beta1.ProviderSpec) (*machinev1beta1.ProviderSpec, error) {
	providerSpec := &machinev1beta1.VSphereMachineProviderSpec{}

	if err := json.Unmarshal(ps.Value.Raw, providerSpec); err != nil {
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

	if err := json.Unmarshal(ps.Value.Raw, providerSpec); err != nil {
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

	if err := json.Unmarshal(ps.Value.Raw, providerSpec); err != nil {
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
