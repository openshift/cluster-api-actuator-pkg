package e2e

import (
	"context"

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

var _ = Describe("Unified MachineSet creation on aws", testframework.LabelDisruptive, Ordered, func() {
	var framework *unified.UnifiedFramework
	var cl runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var helper *TestHelper

	BeforeAll(func() {
		var err error
		framework = unified.NewUnifiedFramework()
		cl, err = testframework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Should load client")
		komega.SetClient(cl)
		ctx = testframework.GetContext()
		platform, err = testframework.GetPlatform(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Should get platform")

		helper = NewTestHelper(ctx, framework, cl, platform, nil)
		helper.SkipIfNotPlatform(configv1.AWSPlatformType)
	})

	It("creates a new MachineSet via unified backend", func() {
		template := helper.CreateTemplate("unified-machineset-template")
		defer helper.DeleteTemplate(template)
		machineSet := helper.CreateMachineSet("unified-machineset", template, nil)
		defer helper.DeleteMachineSet(machineSet)
		Expect(framework.WaitForMachinesRunning(ctx, cl, machineSet)).To(Succeed(), "Should have machines running")
	})

	It("creates a spot instance MachineSet via unified backend", func() {
		// Create spot configuration
		spotConfig := &config.MachineTemplateConfig{
			AWS: &config.AWSMachineConfig{
				SpotMarketOptions: &config.SpotMarketConfig{
					MaxPrice: nil, // Use default price.
				},
				Tenancy: func() *string { s := "default"; return &s }(),
			},
		}

		// Create template with spot configuration applied during creation.
		template, err := framework.CreateMachineTemplate(ctx, cl, platform, backends.BackendMachineTemplateParams{
			Name:     "spot-template",
			Platform: platform,
			Spec:     spotConfig, // Pass configuration during template creation
		})
		Expect(err).NotTo(HaveOccurred(), "Should create Machine Template with spot configuration")
		defer helper.DeleteTemplate(template)

		machineSet := helper.CreateMachineSet("spot-machineset", template, nil)
		defer helper.DeleteMachineSet(machineSet)
		Expect(framework.WaitForMachinesRunning(ctx, cl, machineSet)).To(Succeed(), "Should have machines running")

		By("Verifying spot instance configuration contains spot-related fields")
		helper.VerifyMachineSetContainsString(machineSet, "spot")
	})
})
