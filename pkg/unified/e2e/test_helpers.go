package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	corev1 "k8s.io/api/core/v1"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

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
	status, err := helper.framework.GetMachineSetStatus(helper.ctx, helper.client, machineSet)
	Expect(err).NotTo(HaveOccurred(), "Should get MachineSet status")
	Expect(status.Replicas).To(BeNumerically(">=", 0))
}

// WaitForMachinesRunningOrSkipOnCapacityError waits for machines to become running,
// but skips the test if InsufficientInstanceCapacity error is detected.
func (helper *TestHelper) WaitForMachinesRunningOrSkipOnCapacityError(machineSet interface{}) {
	Eventually(func() error {
		// First check if there's a capacity error
		machineSetJSON, err := json.Marshal(machineSet)
		if err == nil {
			machineSetStr := string(machineSetJSON)
			if strings.Contains(machineSetStr, "InsufficientInstanceCapacity") {
				Skip("Skipping test due to InsufficientInstanceCapacity - AWS does not have sufficient capacity in the requested Availability Zone")
			}
		}

		// Then try to wait for machines to be running
		return helper.framework.WaitForMachinesRunning(helper.ctx, helper.client, machineSet)
	}, framework.WaitLong, framework.RetryMedium).Should(Succeed(), "Should have machines running or skip on capacity error")
}

// DeleteTemplate safely deletes a machine template.
func (helper *TestHelper) DeleteTemplate(template interface{}) {
	err := helper.framework.DeleteMachineTemplate(helper.ctx, helper.client, template)
	Expect(err).NotTo(HaveOccurred(), "Should delete machine template")
}

// DeleteMachineSet safely deletes a machine set.
func (helper *TestHelper) DeleteMachineSet(machineSet interface{}) {
	err := helper.framework.DeleteMachineSet(helper.ctx, helper.client, machineSet)
	Expect(err).NotTo(HaveOccurred(), "Should delete MachineSet")
}

// SkipIfNotPlatform skips the test if the platform does not match the required platform.
func (helper *TestHelper) SkipIfNotPlatform(requiredPlatform configv1.PlatformType) {
	if helper.platform != requiredPlatform {
		Skip(fmt.Sprintf("These features are only supported on %s platform", requiredPlatform))
	}
}

// VerifyMachineSetContainsString verifies that MachineSet contains specific strings (fuzzy matching).
func (helper *TestHelper) VerifyMachineSetContainsString(machineSet interface{}, searchStrings ...string) {
	switch machineSetType := machineSet.(type) {
	case *machinev1.MachineSet:
		helper.verifyMAPIMachineSetContainsString(machineSetType, searchStrings...)
	case *clusterv1.MachineSet:
		helper.verifyCAPIMachineSetContainsString(machineSetType, searchStrings...)
	default:
		Fail(fmt.Sprintf("Unsupported MachineSet type: %T", machineSet))
	}
}

// verifyMAPIMachineSetContainsString verifies MAPI MachineSet contains specific strings.
func (helper *TestHelper) verifyMAPIMachineSetContainsString(machineSet *machinev1.MachineSet, searchStrings ...string) {
	if machineSet.Spec.Template.Spec.ProviderSpec.Value == nil {
		Fail("MAPI MachineSet ProviderSpec is nil")
	}

	// Convert entire ProviderSpec to string for searching.
	configBytes := machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw
	configString := string(configBytes)

	for _, searchString := range searchStrings {
		Expect(configString).To(ContainSubstring(searchString),
			fmt.Sprintf("MAPI MachineSet should contain string '%s' in ProviderSpec", searchString))
	}
}

// verifyCAPIMachineSetContainsString verifies CAPI MachineSet contains specific strings.
func (helper *TestHelper) verifyCAPIMachineSetContainsString(machineSet *clusterv1.MachineSet, searchStrings ...string) {
	// For CAPI, we need to check the associated MachineTemplate (AWSMachineTemplate).
	infraRef := machineSet.Spec.Template.Spec.InfrastructureRef
	if infraRef.Kind != "AWSMachineTemplate" {
		Fail(fmt.Sprintf("Expected AWSMachineTemplate, got %s", infraRef.Kind))
	}

	// Get the AWSMachineTemplate.
	awsTemplate := &awsv1.AWSMachineTemplate{}
	err := helper.client.Get(helper.ctx, runtimeclient.ObjectKey{
		Name:      infraRef.Name,
		Namespace: infraRef.Namespace,
	}, awsTemplate)
	Expect(err).NotTo(HaveOccurred(), "Should get AWSMachineTemplate %s/%s", infraRef.Namespace, infraRef.Name)

	// Search in the entire AWSMachineTemplate object instead of just Spec.Template.Spec.
	fullTemplateBytes, err := json.Marshal(awsTemplate)
	Expect(err).NotTo(HaveOccurred(), "Should marshal full AWSMachineTemplate")

	configString := string(fullTemplateBytes)

	for _, searchString := range searchStrings {
		Expect(configString).To(ContainSubstring(searchString),
			fmt.Sprintf("CAPI AWSMachineTemplate should contain string '%s' anywhere in template", searchString))
	}
}

// StringPtr returns a pointer to the string value.
func StringPtr(s string) *string {
	return &s
}

// GetRegion retrieves the region from node labels for any cloud platform.
func (helper *TestHelper) GetRegion() string {
	// Get the first node to read the region label
	nodeList := &corev1.NodeList{}
	err := helper.client.List(helper.ctx, nodeList, runtimeclient.Limit(1))
	Expect(err).NotTo(HaveOccurred(), "Should list nodes")

	Expect(nodeList.Items).NotTo(BeEmpty(), "Should have at least one node in the cluster")

	// Get region from the topology label
	node := nodeList.Items[0]
	region, ok := node.Labels["topology.kubernetes.io/region"]
	Expect(ok).To(BeTrue(), "Should have topology.kubernetes.io/region label on node %s", node.Name)
	Expect(region).NotTo(BeEmpty(), "Should have non-empty region label on node %s", node.Name)

	return region
}
