package unified

import (
	"context"

	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/backends"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
)

// UnifiedFramework provides a unified testing framework interface.
type UnifiedFramework struct {
	config  *config.TestConfig
	backend backends.MachineBackend
}

// NewUnifiedFramework creates a new unified testing framework.
func NewUnifiedFramework() *UnifiedFramework {
	testConfig := config.LoadTestConfig()

	backend, err := backends.NewBackend(
		backends.BackendType(testConfig.BackendType),
		backends.BackendType(testConfig.AuthoritativeAPI),
	)
	Expect(err).NotTo(HaveOccurred(), "Should create backend")

	return &UnifiedFramework{
		config:  testConfig,
		backend: backend,
	}
}

// GetBackendType returns the backend type.
func (framework *UnifiedFramework) GetBackendType() backends.BackendType {
	return backends.BackendType(framework.config.BackendType)
}

// GetAuthoritativeAPI returns the authoritative API type.
func (framework *UnifiedFramework) GetAuthoritativeAPI() backends.BackendType {
	return backends.BackendType(framework.config.AuthoritativeAPI)
}

// CreateMachineSet creates a machine set.
func (framework *UnifiedFramework) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params backends.BackendMachineSetParams) (interface{}, error) {
	return framework.backend.CreateMachineSet(ctx, client, params)
}

// DeleteMachineSet deletes a machine set.
func (framework *UnifiedFramework) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return framework.backend.DeleteMachineSet(ctx, client, machineSet)
}

// WaitForMachineSetDeleted waits for machine set deletion.
func (framework *UnifiedFramework) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return framework.backend.WaitForMachineSetDeleted(ctx, client, machineSet)
}

// WaitForMachinesRunning waits for all machines belonging to the machine set to enter the "Running" phase.
func (framework *UnifiedFramework) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	return framework.backend.WaitForMachinesRunning(ctx, client, machineSet)
}

// GetMachineSetStatus returns the machine set status.
func (framework *UnifiedFramework) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*backends.MachineSetStatus, error) {
	return framework.backend.GetMachineSetStatus(ctx, client, machineSet)
}

// GetNodesFromMachineSet returns nodes from a machine set.
func (framework *UnifiedFramework) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	return framework.backend.GetNodesFromMachineSet(ctx, client, machineSet)
}

// CreateMachineTemplate creates a machine template.
func (framework *UnifiedFramework) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params backends.BackendMachineTemplateParams) (interface{}, error) {
	return framework.backend.CreateMachineTemplate(ctx, client, platform, params)
}

// DeleteMachineTemplate deletes a machine template.
func (framework *UnifiedFramework) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	return framework.backend.DeleteMachineTemplate(ctx, client, template)
}
