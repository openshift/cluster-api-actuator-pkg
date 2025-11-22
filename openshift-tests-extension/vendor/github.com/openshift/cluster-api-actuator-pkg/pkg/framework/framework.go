package framework

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	cov1helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"github.com/tidwall/gjson"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

// Various constants used by E2E tests.
const (
	PollNodesReadyTimeout   = 10 * time.Minute
	ClusterKey              = "machine.openshift.io/cluster-api-cluster"
	MachineSetKey           = "machine.openshift.io/cluster-api-machineset"
	MachineAPINamespace     = "openshift-machine-api"
	ClusterAPINamespace     = "openshift-cluster-api"
	GlobalInfrastuctureName = "cluster"
	WorkerNodeRoleLabel     = "node-role.kubernetes.io/worker"
	RetryShort              = 1 * time.Second
	RetryMedium             = 5 * time.Second
	// DefaultMachineSetReplicas is the default number of replicas of a machineset
	// if MachineSet.Spec.Replicas field is set to nil.
	DefaultMachineSetReplicas  = 0
	MachinePhaseRunning        = "Running"
	MachinePhaseFailed         = "Failed"
	MachineRoleLabel           = "machine.openshift.io/cluster-api-machine-role"
	MachineTypeLabel           = "machine.openshift.io/cluster-api-machine-type"
	MachineAnnotationKey       = "machine.openshift.io/machine"
	ClusterAPIActuatorPkgTaint = "cluster-api-actuator-pkg"

	// Openshift CI specific env variables.
	isCI        = "OPENSHIFT_CI"
	artifactDir = "ARTIFACT_DIR"
	cliDir      = "CLI_DIR"
)

var (
	WaitShort      = 1 * time.Minute
	WaitMedium     = 3 * time.Minute
	WaitOverMedium = 5 * time.Minute
	WaitLong       = 15 * time.Minute
	WaitOverLong   = 30 * time.Minute
)

// GetInfrastructure fetches the global cluster infrastructure object.
func GetInfrastructure(ctx context.Context, c runtimeclient.Client) (*configv1.Infrastructure, error) {
	infra := &configv1.Infrastructure{}
	infraName := runtimeclient.ObjectKey{
		Name: GlobalInfrastuctureName,
	}

	if err := c.Get(ctx, infraName, infra); err != nil {
		return nil, err
	}

	return infra, nil
}

var platform configv1.PlatformType

// GetPlatform fetches the PlatformType from the infrastructure object.
// Caches value after first successful retrieval.
func GetPlatform(ctx context.Context, c runtimeclient.Client) (configv1.PlatformType, error) {
	// platform won't change during test run and might be cached
	if platform != "" {
		return platform, nil
	}

	infra, err := GetInfrastructure(ctx, c)
	if err != nil {
		return "", err
	}

	if infra.Status.PlatformStatus == nil {
		return "", errors.New("platform status is not populated in infrastructure object")
	}

	platform = infra.Status.PlatformStatus.Type

	return platform, nil
}

// LoadClient returns a new controller-runtime client.
func LoadClient() (runtimeclient.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return runtimeclient.New(cfg, runtimeclient.Options{})
}

// LoadClientset returns a new Kubernetes Clientset.
func LoadClientset() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

// GetContext returns a context.
func GetContext() context.Context {
	return context.Background()
}

func WaitForStatusAvailableShort(ctx context.Context, client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(ctx, client, name, WaitShort)
}

func WaitForStatusAvailableMedium(ctx context.Context, client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(ctx, client, name, WaitMedium)
}

func WaitForStatusAvailableOverLong(ctx context.Context, client runtimeclient.Client, name string) bool {
	return expectStatusAvailableIn(ctx, client, name, WaitOverLong)
}

func expectStatusAvailableIn(ctx context.Context, client runtimeclient.Client, name string, timeout time.Duration) bool {
	key := types.NamespacedName{
		Name: name,
	}
	clusterOperator := &configv1.ClusterOperator{}

	if err := wait.PollUntilContextTimeout(ctx, RetryMedium, timeout, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, key, clusterOperator); err != nil {
			klog.Errorf("error querying api for OperatorStatus object: %v, retrying...", err)
			return false, nil
		}
		if cov1helpers.IsStatusConditionFalse(clusterOperator.Status.Conditions, configv1.OperatorAvailable) {
			klog.Errorf("Condition: %q is false", configv1.OperatorAvailable)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorProgressing) {
			klog.Errorf("Condition: %q is true", configv1.OperatorProgressing)
			return false, nil
		}
		if cov1helpers.IsStatusConditionTrue(clusterOperator.Status.Conditions, configv1.OperatorDegraded) {
			klog.Errorf("Condition: %q is true", configv1.OperatorDegraded)
			return false, nil
		}

		return true, nil
	}); err != nil {
		klog.Errorf("Error checking isStatusAvailable: %v", err)
		return false
	}

	return true
}

func WaitForValidatingWebhook(ctx context.Context, client runtimeclient.Client, name string) bool {
	key := types.NamespacedName{Name: name}
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := client.Get(ctx, key, webhook); err != nil {
			klog.Errorf("error querying api for ValidatingWebhookConfiguration: %v, retrying...", err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		klog.Errorf("Error waiting for ValidatingWebhookConfiguration: %v", err)
		return false
	}

	return true
}

// WaitForEvent expects to find the given event.
func WaitForEvent(ctx context.Context, c runtimeclient.Client, kind, name, reason string) error {
	return wait.PollUntilContextTimeout(ctx, RetryMedium, WaitMedium, true, func(ctx context.Context) (bool, error) {
		eventList := corev1.EventList{}
		if err := c.List(ctx, &eventList); err != nil {
			klog.Errorf("error querying api for eventList object: %v, retrying...", err)
			return false, nil
		}

		for _, event := range eventList.Items {
			if event.Reason != reason ||
				event.InvolvedObject.Kind != kind ||
				event.InvolvedObject.Name != name {
				continue
			}

			return true, nil
		}

		return false, nil
	})
}

// NewCLI initializes oc binary wrapper helper.
// Output and oc executable path configure depending on the environment.
// If Openshift CI is detected, respective parameters are set up.
func NewCLI() (*gatherer.CLI, error) {
	client, err := LoadClient()
	if err != nil {
		return nil, err
	}

	baseOutputPath, err := getCliOutputFilesPath()
	if err != nil {
		return nil, err
	}

	cli, err := gatherer.NewCLI(MachineAPINamespace, client, baseOutputPath)
	if err != nil {
		return nil, err
	}

	cli = cli.WithExec(getOcExecPath())

	return cli, nil
}

// isOpenshiftCI tries to detect Openshift CI environment.
func isOpenshiftCI() bool {
	envCI := os.Getenv(isCI)
	envArtifactsDir := os.Getenv(artifactDir)

	return envCI == "true" && len(envArtifactsDir) > 0
}

// getCliOutputFilesPath returns output path for the CLI wrapper.
// In case Openshift CI env detected, returns '$ARTIFACT_DIR/machine-api-e2e-suite' path.
// If not, returns '%current_directory%/_out'.
func getCliOutputFilesPath() (string, error) {
	if isOpenshiftCI() {
		return filepath.Join(os.Getenv(artifactDir), "machine-api-e2e-suite"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	return filepath.Join(cwd, "_out"), nil
}

func getOcExecPath() string {
	if isOpenshiftCI() {
		return filepath.Join(os.Getenv(cliDir), "oc")
	}

	return "oc"
}

// NewGatherer initializes StateGatherer - helper for collection of MAPI-related resources and pod logs in tests.
func NewGatherer() (*gatherer.StateGatherer, error) {
	cli, err := NewCLI()
	if err != nil {
		return nil, err
	}

	return gatherer.NewStateGatherer(context.Background(), cli, time.Now()), nil
}

// DeleteObjects deletes the objects in the given list.
func DeleteObjects(ctx context.Context, cl runtimeclient.Client, objs ...runtimeclient.Object) {
	for _, o := range objs {
		By(fmt.Sprintf("Deleting %s/%s", o.GetObjectKind().GroupVersionKind().Kind, o.GetName()))
		Expect(cl.Delete(ctx, o)).To(Succeed())
	}
}

// SkipIfNotTechPreviewNoUpgrade skip test if a cluster is not a TechPreviewNoUpgrade cluster.
func SkipIfNotTechPreviewNoUpgrade(oc *gatherer.CLI, cl runtimeclient.Client) {
	featureSet, err := oc.WithoutNamespace().Run("get").Args("featuregate", "cluster", "-o=jsonpath={.spec.featureSet}").Output()
	Expect(err).NotTo(HaveOccurred(), "Failed to get featureSet")

	if featureSet != string(configv1.TechPreviewNoUpgrade) {
		Skip("FeatureSet is not TechPreviewNoUpgradec, skip it!")
	}
}

// GetCredentialsFromCluster get credentials from cluster.
func GetCredentialsFromCluster(oc *gatherer.CLI) ([]byte, []byte, string) {
	awscreds, err := oc.WithoutNamespace().Run("get").Args("secret/aws-creds", "-n", "kube-system", "-o", "json").Output()
	if err != nil {
		Skip("Unable to get AWS credentials secret, skipping the testing.")
	}

	accessKeyIDBase64, secureKeyBase64 := gjson.Get(awscreds, `data.aws_access_key_id`).String(), gjson.Get(awscreds, `data.aws_secret_access_key`).String()

	accessKeyID, err := base64.StdEncoding.DecodeString(accessKeyIDBase64)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode accessKeyID")
	secureKey, err := base64.StdEncoding.DecodeString(secureKeyBase64)
	Expect(err).NotTo(HaveOccurred(), "Failed to decode secureKey")
	clusterRegion, err := oc.WithoutNamespace().Run("get").Args("infrastructure", "cluster", "-o=jsonpath={.status.platformStatus.aws.region}").Output()
	Expect(err).NotTo(HaveOccurred(), "Failed to get clusterRegion")

	return accessKeyID, secureKey, clusterRegion
}

// IsCustomerVPC check if cluster is customer vpc cluster.
func IsCustomerVPC(oc *gatherer.CLI) bool {
	installConfig, err := oc.WithoutNamespace().Run("get").Args("cm", "cluster-config-v1", "-n", "kube-system", "-o=jsonpath={.data.install-config}").Output()
	Expect(err).NotTo(HaveOccurred(), "Failed to get install-config")

	switch platform {
	case configv1.AWSPlatformType:
		return strings.Contains(installConfig, "subnets:")
	case configv1.AzurePlatformType:
		return strings.Contains(installConfig, "virtualNetwork:")
	default:
		return false
	}
}
