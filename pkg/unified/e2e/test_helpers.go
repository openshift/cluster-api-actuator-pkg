package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
	corev1 "k8s.io/api/core/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Infrastructure reference kinds.
	awsMachineKind         = "AWSMachine"
	awsMachineTemplateKind = "AWSMachineTemplate"

	// Kubernetes topology labels.
	topologyRegionLabel = "topology.kubernetes.io/region"
)

// generateName returns a unique resource name by appending a random suffix to
// the given prefix. This avoids name collisions between Ordered test contexts
// that run sequentially on the same cluster.
func generateName(prefix string) string {
	return prefix + utilrand.String(5)
}

// TestHelper provides common helper functions for unified framework tests.
type TestHelper struct {
	framework   *unified.UnifiedFramework
	client      runtimeclient.Client
	ctx         context.Context
	platform    configv1.PlatformType
	machineSpec interface{}
}

// NewTestHelper creates a new test helper instance.
func NewTestHelper(ctx context.Context, framework *unified.UnifiedFramework, client runtimeclient.Client, platform configv1.PlatformType, machineSpec interface{}) *TestHelper {
	return &TestHelper{
		framework:   framework,
		client:      client,
		ctx:         ctx,
		platform:    platform,
		machineSpec: machineSpec,
	}
}

// CreateTemplate creates and validates a machine template.
func (helper *TestHelper) CreateTemplate(name string) interface{} {
	GinkgoHelper()

	template, err := helper.framework.CreateMachineTemplate(helper.ctx, helper.client, helper.platform, backends.BackendMachineTemplateParams{
		Name:     name,
		Platform: helper.platform,
		Spec:     helper.machineSpec,
	})
	Expect(err).NotTo(HaveOccurred(), "Should create Machine Template")

	return template
}

// CreateMachineSet creates and validates a machine set.
func (helper *TestHelper) CreateMachineSet(name string, template interface{}, labels map[string]string) interface{} {
	GinkgoHelper()

	machineSet, err := helper.framework.CreateMachineSet(helper.ctx, helper.client, backends.BackendMachineSetParams{
		Name:             name,
		Replicas:         1,
		Labels:           labels,
		Annotations:      map[string]string{"e2e": name},
		Template:         template,
		FailureDomain:    "auto",
		AuthoritativeAPI: helper.framework.GetAuthoritativeAPI(), // Use the framework's authoritative API setting
	})
	Expect(err).NotTo(HaveOccurred(), "Should create MachineSet")

	return machineSet
}

// ValidateStatus validates machine set status.
func (helper *TestHelper) ValidateStatus(machineSet interface{}) {
	GinkgoHelper()

	status, err := helper.framework.GetMachineSetStatus(helper.ctx, helper.client, machineSet)
	Expect(err).NotTo(HaveOccurred(), "Should get MachineSet status")
	Expect(status.Replicas).To(BeNumerically(">=", 0), "Should have valid replica count")
}

// WaitForMachinesRunningOrSkipOnCapacityError waits for machines to become running,
// but skips the test if InsufficientInstanceCapacity error is detected.
func (helper *TestHelper) WaitForMachinesRunningOrSkipOnCapacityError(machineSet interface{}) {
	GinkgoHelper()

	var err error

	switch helper.framework.GetBackendType() {
	case config.BackendTypeMAPI:
		ms, ok := machineSet.(*machinev1.MachineSet)
		Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)
		// WaitForSpotMachineSet re-fetches Machines each poll cycle and inspects
		// per-machine ProviderStatus for capacity conditions.
		err = framework.WaitForSpotMachineSet(helper.ctx, helper.client, ms.Name)
	case config.BackendTypeCAPI:
		ms, ok := machineSet.(*clusterv1beta1.MachineSet)
		Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)
		// WaitForCAPIMachinesRunningWithRetry re-fetches InfraMachines each poll
		// cycle and searches their status for the given error keys.
		err = framework.WaitForCAPIMachinesRunningWithRetry(helper.ctx, helper.client, ms.Name,
			[]string{"InsufficientInstanceCapacity"})
	default:
		Expect(false).To(BeTrue(), "Should have supported backend type, got %s", helper.framework.GetBackendType())
	}

	if errors.Is(err, framework.ErrMachineNotProvisionedInsufficientCloudCapacity) {
		Skip("Skipping test: insufficient cloud provider capacity in the requested Availability Zone")
	}

	Expect(err).NotTo(HaveOccurred(), "Should have machines running")
}

// DeleteTemplate safely deletes a machine template.
func (helper *TestHelper) DeleteTemplate(template interface{}) {
	GinkgoHelper()

	err := helper.framework.DeleteMachineTemplate(helper.ctx, helper.client, template)
	Expect(err).NotTo(HaveOccurred(), "Should delete machine template")
}

// DeleteMachineSet safely deletes a machine set.
func (helper *TestHelper) DeleteMachineSet(machineSet interface{}) {
	GinkgoHelper()

	err := helper.framework.DeleteMachineSet(helper.ctx, helper.client, machineSet)
	Expect(err).NotTo(HaveOccurred(), "Should delete MachineSet")
}

// SkipIfNotPlatform skips the test if the platform does not match the required platform.
func (helper *TestHelper) SkipIfNotPlatform(requiredPlatform configv1.PlatformType) {
	GinkgoHelper()

	if helper.platform != requiredPlatform {
		Skip(fmt.Sprintf("These features are only supported on %s platform", requiredPlatform))
	}
}

// VerifyMachineSetContainsString verifies that MachineSet contains specific strings (fuzzy matching).
func (helper *TestHelper) VerifyMachineSetContainsString(machineSet interface{}, searchStrings ...string) {
	GinkgoHelper()

	switch machineSetType := machineSet.(type) {
	case *machinev1.MachineSet:
		helper.verifyMAPIMachineSetContainsString(machineSetType, searchStrings...)
	case *clusterv1beta1.MachineSet:
		helper.verifyCAPIMachineSetContainsString(machineSetType, searchStrings...)
	default:
		Expect(false).To(BeTrue(), "Should have supported MachineSet type, got %T", machineSet)
	}
}

// verifyMAPIMachineSetContainsString verifies MAPI MachineSet contains specific strings.
func (helper *TestHelper) verifyMAPIMachineSetContainsString(machineSet *machinev1.MachineSet, searchStrings ...string) {
	GinkgoHelper()

	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).NotTo(BeNil(), "Should have ProviderSpec in MAPI MachineSet")

	// Convert entire ProviderSpec to string for searching.
	configBytes := machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw
	configString := string(configBytes)

	for _, searchString := range searchStrings {
		Expect(configString).To(ContainSubstring(searchString),
			fmt.Sprintf("Should contain string '%s' in MAPI MachineSet ProviderSpec", searchString))
	}
}

// verifyCAPIMachineSetContainsString verifies CAPI MachineSet contains specific strings.
func (helper *TestHelper) verifyCAPIMachineSetContainsString(machineSet *clusterv1beta1.MachineSet, searchStrings ...string) {
	GinkgoHelper()

	// For CAPI, we need to check the associated MachineTemplate (AWSMachineTemplate).
	infraRef := machineSet.Spec.Template.Spec.InfrastructureRef
	Expect(infraRef.Kind).To(Equal(awsMachineTemplateKind), "Should have %s infrastructure reference, got %s", awsMachineTemplateKind, infraRef.Kind)

	// Get the AWSMachineTemplate.
	awsTemplate := &awsv1.AWSMachineTemplate{}
	err := helper.client.Get(helper.ctx, runtimeclient.ObjectKey{
		Name:      infraRef.Name,
		Namespace: infraRef.Namespace,
	}, awsTemplate)
	Expect(err).NotTo(HaveOccurred(), "Should get AWSMachineTemplate %s/%s", infraRef.Namespace, infraRef.Name)

	machineSpecBytes, err := json.Marshal(awsTemplate.Spec.Template.Spec)
	Expect(err).NotTo(HaveOccurred(), "Should marshal AWSMachineTemplate Spec.Template.Spec")

	configString := string(machineSpecBytes)

	for _, searchString := range searchStrings {
		Expect(configString).To(ContainSubstring(searchString),
			fmt.Sprintf("Should contain string '%s' in CAPI AWSMachineTemplate Spec.Template.Spec", searchString))
	}
}

// GetRegion retrieves the region from node labels for any cloud platform.
func (helper *TestHelper) GetRegion() string {
	GinkgoHelper()

	// Get the first node to read the region label
	nodeList := &corev1.NodeList{}
	err := helper.client.List(helper.ctx, nodeList, runtimeclient.Limit(1))
	Expect(err).NotTo(HaveOccurred(), "Should list nodes")

	Expect(nodeList.Items).NotTo(BeEmpty(), "Should have at least one node in the cluster")

	// Get region from the topology label
	node := nodeList.Items[0]
	region, ok := node.Labels[topologyRegionLabel]
	Expect(ok).To(BeTrue(), "Should have %s label on node %s", topologyRegionLabel, node.Name)
	Expect(region).NotTo(BeEmpty(), "Should have non-empty region label on node %s", node.Name)

	return region
}
