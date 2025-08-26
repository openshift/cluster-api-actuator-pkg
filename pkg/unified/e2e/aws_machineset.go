package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Unified MachineSet creation on aws", framework.LabelDisruptive, Ordered, func() {
	var uf *unified.UnifiedFramework
	var cl runtimeclient.Client
	var ctx context.Context
	var platform configv1.PlatformType
	var helper *TestHelper

	BeforeAll(func() {
		var err error
		uf = unified.NewUnifiedFramework()
		cl, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Should load client")
		komega.SetClient(cl)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, cl)
		Expect(err).NotTo(HaveOccurred(), "Should get platform")

		helper = NewTestHelper(ctx, uf, cl, platform, nil)
		helper.SkipIfNotPlatform(configv1.AWSPlatformType)
	})

	It("creates a new MachineSet via unified backend", func() {
		tpl := helper.CreateTemplate("unified-ms-template")
		defer helper.DeleteTemplate(tpl)
		ms := helper.CreateMachineSet("unified-ms", tpl, nil)
		defer helper.DeleteMachineSet(ms)
		Expect(uf.WaitForMachinesRunning(ctx, cl, ms)).To(Succeed(), "Should have machines running")
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
		tpl, err := uf.CreateMachineTemplate(ctx, cl, platform, backends.BackendMachineTemplateParams{
			Name:     "unified-spot-template",
			Platform: platform,
			Spec:     spotConfig, // Pass configuration during template creation
		})
		Expect(err).NotTo(HaveOccurred(), "Should create Machine Template with spot configuration")
		defer helper.DeleteTemplate(tpl)

		ms := helper.CreateMachineSet("unified-spot-ms", tpl, nil)
		defer helper.DeleteMachineSet(ms)
		Expect(uf.WaitForMachinesRunning(ctx, cl, ms)).To(Succeed(), "Should have machines running")

		By("Verifying spot instance configuration contains spot-related fields")
		helper.VerifyMachineSetContainsString(ms, "spot")
	})
})
