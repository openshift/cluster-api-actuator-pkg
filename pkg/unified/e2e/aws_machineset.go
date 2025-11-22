package e2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	testframework "github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe(fmt.Sprintf("MachineSet creation on AWS (backend: %s, authority: %s)",
	config.LoadTestConfig().BackendType,
	config.LoadTestConfig().AuthoritativeAPI), testframework.LabelDisruptive, testframework.LabelUnified, Ordered, func() {
	var framework *unified.UnifiedFramework
	var cl runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var helper *TestHelper

	BeforeAll(func() {
		var err error
		framework = unified.NewUnifiedFramework()
		By(fmt.Sprintf("Framework initialized with backend: %s, authoritativeAPI: %s", framework.GetBackendType(), framework.GetAuthoritativeAPI()))

		cl, err = testframework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Should load client")
		komega.SetClient(cl)
		ctx = testframework.GetContext()
		platform, err = testframework.GetPlatform(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Should get platform")

		helper = NewTestHelper(ctx, framework, cl, platform, nil)
		helper.SkipIfNotPlatform(configv1.AWSPlatformType)
	})

	It("creates a new MachineSet", func() {
		By("Creating a MachineTemplate")
		template := helper.CreateTemplate("unified-machineset-template")
		defer helper.DeleteTemplate(template)

		By("Creating a MachineSet from the template")
		machineSet := helper.CreateMachineSet("unified-machineset", template, nil)
		defer helper.DeleteMachineSet(machineSet)

		By("Waiting for Machines to become Running")
		Expect(framework.WaitForMachinesRunning(ctx, cl, machineSet)).To(Succeed(), "Should have machines running")
	})

	It("creates a spot instance MachineSet", func() {
		By("Creating a MachineTemplate with spot instance configuration")
		spotConfig := &config.MachineTemplateConfig{
			AWS: &config.AWSMachineConfig{
				SpotMarketOptions: &config.SpotMarketConfig{
					MaxPrice: nil, // Use default price.
				},
				Tenancy: StringPtr("default"),
			},
		}

		template, err := framework.CreateMachineTemplate(ctx, cl, platform, backends.BackendMachineTemplateParams{
			Name:     "spot-template",
			Platform: platform,
			Spec:     spotConfig,
		})
		Expect(err).NotTo(HaveOccurred(), "Should create Machine Template with spot configuration")
		defer helper.DeleteTemplate(template)

		By("Creating a MachineSet from the spot template")
		machineSet := helper.CreateMachineSet("spot-machineset", template, nil)
		defer helper.DeleteMachineSet(machineSet)

		By("Waiting for Machines to become Running or skipping on capacity error")
		helper.WaitForMachinesRunningOrSkipOnCapacityError(machineSet)

		By("Verifying spot instance configuration is correctly applied")
		helper.VerifyMachineSetContainsString(machineSet, "spot")
	})

	It("creates a MachineSet with EFA network interface type", func() {
		By("Checking if region supports EFA")
		region := helper.GetRegion()

		if region != "us-east-2" && region != "us-west-2" {
			Skip(fmt.Sprintf("EFA test is only supported in us-east-2 and us-west-2, current region: %s", region))
		}

		By("Creating a MachineTemplate with EFA network interface and c5n.9xlarge instance type")
		efaConfig := &config.MachineTemplateConfig{
			AWS: &config.AWSMachineConfig{
				InstanceType:         StringPtr("c5n.9xlarge"),
				NetworkInterfaceType: StringPtr("efa"),
			},
		}

		template, err := framework.CreateMachineTemplate(ctx, cl, platform, backends.BackendMachineTemplateParams{
			Name:     "efa-template",
			Platform: platform,
			Spec:     efaConfig,
		})
		Expect(err).NotTo(HaveOccurred(), "Should create Machine Template with EFA network interface configuration")
		defer helper.DeleteTemplate(template)

		By("Creating a MachineSet from the EFA template")
		machineSet := helper.CreateMachineSet("efa-machineset", template, nil)
		defer helper.DeleteMachineSet(machineSet)

		By("Waiting for Machines to become Running or skipping on capacity error")
		helper.WaitForMachinesRunningOrSkipOnCapacityError(machineSet)

		By("Verifying EFA network interface configuration is correctly applied")
		// MAPI uses "EFA", CAPI uses "efa"
		if framework.GetBackendType() == "MAPI" {
			helper.VerifyMachineSetContainsString(machineSet, "EFA")
		} else {
			helper.VerifyMachineSetContainsString(machineSet, "efa")
		}
	})
})
