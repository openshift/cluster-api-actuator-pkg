package infra

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("[sig-cluster-lifecycle] Machine API GCP Disk Licenses", framework.LabelMAPI, framework.LabelDisruptive, framework.LabelPeriodic, func() {
	var ctx context.Context

	var (
		client     runtimeclient.Client
		machineSet *machinev1.MachineSet
	)

	var stateGatherer *gatherer.StateGatherer

	BeforeEach(func() {
		var err error

		ctx = framework.GetContext()

		stateGatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "StateGatherer should be able to be created")

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Controller-runtime client should be able to be created")

		// Only run on GCP clusters.
		platform, err := framework.GetPlatform(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Should be able to get Platform type")

		if platform != configv1.GCPPlatformType {
			Skip(fmt.Sprintf("Platform %s is not GCP, skipping GCP disk license test.", platform))
		}

		// Make sure to clean up the resources we created.
		DeferCleanup(func() {
			if machineSet != nil {
				By("Deleting the MachineSet with disk licenses")
				Expect(client.Delete(ctx, machineSet)).To(Succeed(), "MachineSet should be able to be deleted")
				framework.WaitForMachineSetsDeleted(ctx, client, machineSet)
			}
		})
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(stateGatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "StateGatherer should be able to gather resources")
		}
	})

	// Machines required for test: 1
	// Reason: We create a single Machine with disk licenses to verify the GCP API accepts
	// the Licenses field on InitializeParams and the Machine reaches Running phase.
	It("should create a Machine with licenses on the boot disk [GCP]", func() {
		By("Getting the base MachineSet parameters from a worker MachineSet")

		machineSetParams := framework.BuildMachineSetParams(ctx, client, 1)

		By("Adding licenses to the boot disk in the GCP provider spec")
		Expect(setLicensesOnGCPProviderSpec(machineSetParams, []string{
			"projects/vm-options/global/licenses/enable-vmx",
		})).To(Succeed(), "Should be able to set disk licenses on GCP ProviderSpec")

		var err error

		By("Creating a MachineSet with disk licenses")

		machineSet, err = framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred(), "MachineSet with disk licenses should be able to be created")

		By("Waiting for the Machine to reach Running phase")
		framework.WaitForMachineSet(ctx, client, machineSet.GetName())

		By("Verifying the Machine reached Running phase with disk licenses")

		machines, err := framework.GetMachinesFromMachineSet(ctx, client, machineSet)
		Expect(err).ToNot(HaveOccurred(), "Should be able to get Machines from MachineSet")
		Expect(machines).To(HaveLen(1), "Should have exactly 1 Machine")

		running := framework.FilterRunningMachines(machines)
		Expect(running).To(HaveLen(1), "The Machine with disk licenses should be in Running phase")

		By("Verifying the providerSpec on the Machine includes the expected licenses")

		var machineProviderSpec machinev1.GCPMachineProviderSpec

		Expect(json.Unmarshal(running[0].Spec.ProviderSpec.Value.Raw, &machineProviderSpec)).To(Succeed(),
			"Should be able to unmarshal Machine providerSpec")

		bootDiskFound := false

		for _, disk := range machineProviderSpec.Disks {
			if disk.Boot {
				bootDiskFound = true

				Expect(disk.Licenses).To(ContainElement("projects/vm-options/global/licenses/enable-vmx"),
					"Boot disk should have the enable-vmx license")
			}
		}

		Expect(bootDiskFound).To(BeTrue(), "Should find a boot disk in the Machine's providerSpec")
	})
})

// setLicensesOnGCPProviderSpec adds the specified licenses to the boot disk in the GCP provider spec.
func setLicensesOnGCPProviderSpec(params framework.MachineSetParams, licenses []string) error {
	var spec machinev1.GCPMachineProviderSpec
	if err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec); err != nil {
		return fmt.Errorf("error unmarshalling GCP providerSpec: %w", err)
	}

	bootDiskFound := false

	for i := range spec.Disks {
		if spec.Disks[i].Boot {
			spec.Disks[i].Licenses = licenses
			bootDiskFound = true

			break
		}
	}

	if !bootDiskFound {
		return fmt.Errorf("no boot disk found in GCP providerSpec")
	}

	var err error

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling GCP providerSpec: %w", err)
	}

	return nil
}
