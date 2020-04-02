package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	osconfigv1 "github.com/openshift/api/config/v1"
	caov1alpha1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis"
	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/autoscaler"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/infra"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/machinehealthcheck"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/operators"
)

const junitDirEnvVar = "JUNIT_DIR"

func init() {
	if err := mapiv1beta1.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatal(err)
	}

	if err := caov1alpha1.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatal(err)
	}

	if err := osconfigv1.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatal(err)
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
