package e2e

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"

	osconfigv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	caov1alpha1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis"

	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/autoscaler"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/infra"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/machinehealthcheck"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/operators"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/providers"
)

func init() {
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)

	if err := machinev1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := caov1alpha1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := osconfigv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Suite")
}

var _ = BeforeSuite(func() {
	client, err := framework.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	ctx := framework.GetContext()

	platform, err := framework.GetPlatform(ctx, client)
	Expect(err).ToNot(HaveOccurred())

	// Extend timeouts for slower providers
	switch platform {
	case osconfigv1.AzurePlatformType, osconfigv1.GCPPlatformType, osconfigv1.VSpherePlatformType, osconfigv1.OpenStackPlatformType, osconfigv1.PowerVSPlatformType, osconfigv1.NutanixPlatformType:
		framework.WaitShort = 2 * time.Minute  // Normally 1m
		framework.WaitMedium = 6 * time.Minute // Normally 3m
		framework.WaitLong = 30 * time.Minute  // Normally 15m
	}
})
