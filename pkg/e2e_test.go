package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/klog"

	osconfigv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	caov1alpha1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis"
	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/autoscaler"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/infra"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/machinehealthcheck"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/operators"
)

const junitDirEnvVar = "JUNIT_DIR"

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
	RunSpecsWithDefaultAndCustomReporters(t, "Machine Suite", e2eReporters())
}

func e2eReporters() []Reporter {
	reportDir := os.Getenv(junitDirEnvVar)
	if reportDir != "" {
		// Include `ParallelNode` so tests running in parallel do not overwrite the same file.
		// Include timestamp so test suite can be called multiple times with focus within same CI job
		// without overwriting files.
		junitFileName := fmt.Sprintf("%s/junit_cluster_api_actuator_pkg_e2e_%d_%d.xml", reportDir, time.Now().UnixNano(), config.GinkgoConfig.ParallelNode)
		return []Reporter{reporters.NewJUnitReporter(junitFileName)}
	}
	return []Reporter{}
}

var _ = BeforeSuite(func() {
	client, err := framework.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	platform, err := framework.GetPlatform(client)
	Expect(err).ToNot(HaveOccurred())

	// Extend timeouts for slower providers
	switch platform {
	case osconfigv1.AzurePlatformType, osconfigv1.VSpherePlatformType, osconfigv1.OpenStackPlatformType:
		framework.WaitShort = 2 * time.Minute  // Normally 1m
		framework.WaitMedium = 6 * time.Minute // Normally 3m
		framework.WaitLong = 30 * time.Minute  // Normally 15m
	}
})
