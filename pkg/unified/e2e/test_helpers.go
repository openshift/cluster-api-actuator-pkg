package e2e

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestHelper provides common helper functions for unified framework tests.
type TestHelper struct {
	uf          *unified.UnifiedFramework
	client      runtimeclient.Client
	ctx         context.Context
	platform    configv1.PlatformType
	machineSpec interface{}
}

// NewTestHelper creates a new test helper instance.
func NewTestHelper(ctx context.Context, uf *unified.UnifiedFramework, client runtimeclient.Client, platform configv1.PlatformType, machineSpec interface{}) *TestHelper {
	return &TestHelper{
		uf:          uf,
		client:      client,
		ctx:         ctx,
		platform:    platform,
		machineSpec: machineSpec,
	}
}

// CreateTemplate creates and validates a machine template.
func (th *TestHelper) CreateTemplate(name string) interface{} {
	tpl, err := th.uf.CreateMachineTemplate(th.ctx, th.client, th.platform, backends.BackendMachineTemplateParams{
		Name:     name,
		Platform: th.platform,
		Spec:     th.machineSpec,
	})
	Expect(err).NotTo(HaveOccurred(), "Should create Machine Template")

	return tpl
}

// CreateMachineSet creates and validates a machine set.
func (th *TestHelper) CreateMachineSet(name string, template interface{}, labels map[string]string) interface{} {
	ms, err := th.uf.CreateMachineSet(th.ctx, th.client, backends.BackendMachineSetParams{
		Name:             name,
		Replicas:         1,
		Labels:           labels,
		Annotations:      map[string]string{"e2e": name},
		Template:         template,
		FailureDomain:    "auto",
		AuthoritativeAPI: th.uf.GetAuthoritativeAPI(), // Use the framework's authoritative API setting
	})
	Expect(err).NotTo(HaveOccurred(), "Should create MachineSet")

	return ms
}

// ValidateStatus validates machine set status.
func (th *TestHelper) ValidateStatus(ms interface{}) {
	status, err := th.uf.GetMachineSetStatus(th.ctx, th.client, ms)
	Expect(err).NotTo(HaveOccurred(), "Should get MachineSet status")
	Expect(status.Replicas).To(BeNumerically(">=", 0))
}

// DeleteTemplate safely deletes a machine template.
func (th *TestHelper) DeleteTemplate(template interface{}) {
	err := th.uf.DeleteMachineTemplate(th.ctx, th.client, template)
	Expect(err).NotTo(HaveOccurred(), "Should delete machine template")
}

// DeleteMachineSet safely deletes a machine set.
func (th *TestHelper) DeleteMachineSet(ms interface{}) {
	err := th.uf.DeleteMachineSet(th.ctx, th.client, ms)
	Expect(err).NotTo(HaveOccurred(), "Should delete MachineSet")
}

// SkipIfNotPlatform skips the test if the platform does not match the required platform.
func (th *TestHelper) SkipIfNotPlatform(requiredPlatform configv1.PlatformType) {
	if th.platform != requiredPlatform {
		Skip(fmt.Sprintf("These features are only supported on %s platform", requiredPlatform))
	}
}

// VerifyMachineSetContainsString verifies that MachineSet contains specific strings (fuzzy matching).
func (th *TestHelper) VerifyMachineSetContainsString(ms interface{}, searchStrings ...string) {
	switch msType := ms.(type) {
	case *machinev1.MachineSet:
		th.verifyMAPIMachineSetContainsString(msType, searchStrings...)
	case *clusterv1.MachineSet:
		th.verifyCAPIMachineSetContainsString(msType, searchStrings...)
	default:
		Fail(fmt.Sprintf("Unsupported MachineSet type: %T", ms))
	}
}

// verifyMAPIMachineSetContainsString verifies MAPI MachineSet contains specific strings.
func (th *TestHelper) verifyMAPIMachineSetContainsString(ms *machinev1.MachineSet, searchStrings ...string) {
	if ms.Spec.Template.Spec.ProviderSpec.Value == nil {
		Fail("MAPI MachineSet ProviderSpec is nil")
	}

	// Convert entire ProviderSpec to string for searching.
	configBytes := ms.Spec.Template.Spec.ProviderSpec.Value.Raw
	configString := string(configBytes)

	for _, searchString := range searchStrings {
		Expect(configString).To(ContainSubstring(searchString),
			fmt.Sprintf("MAPI MachineSet should contain string '%s' in ProviderSpec", searchString))
	}
}

// verifyCAPIMachineSetContainsString verifies CAPI MachineSet contains specific strings.
func (th *TestHelper) verifyCAPIMachineSetContainsString(ms *clusterv1.MachineSet, searchStrings ...string) {
	// For CAPI, we need to check the associated MachineTemplate (AWSMachineTemplate).
	infraRef := ms.Spec.Template.Spec.InfrastructureRef
	if infraRef.Kind != "AWSMachineTemplate" {
		Fail(fmt.Sprintf("Expected AWSMachineTemplate, got %s", infraRef.Kind))
	}

	// Get the AWSMachineTemplate.
	awsTemplate := &awsv1.AWSMachineTemplate{}
	err := th.client.Get(th.ctx, runtimeclient.ObjectKey{
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
