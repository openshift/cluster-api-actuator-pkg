package backends

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackendType defines backend types.
type BackendType string

const (
	BackendTypeMAPI BackendType = "MAPI"
	BackendTypeCAPI BackendType = "CAPI"
)

// MachineBackend defines the abstract interface for machine backends.
type MachineBackend interface {
	GetBackendType() BackendType
	GetAuthoritativeAPI() BackendType

	CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error)
	DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error
	CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error)
	DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error
	WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error
	WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error
	GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error)
	GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error)
}

// BackendMachineSetParams defines common parameters for creating machine sets.
type BackendMachineSetParams struct {
	Name          string
	Replicas      int32
	Labels        map[string]string
	Annotations   map[string]string
	Template      interface{}
	FailureDomain string
	// AuthoritativeAPI specifies which API should be authoritative for this MachineSet
	AuthoritativeAPI BackendType
}

// BackendMachineTemplateParams defines common parameters for creating machine templates.
type BackendMachineTemplateParams struct {
	Name     string
	Platform configv1.PlatformType
	Spec     interface{}
}

// MachineSetStatus defines common structure for machine set status.
type MachineSetStatus struct {
	Replicas          int32
	AvailableReplicas int32
	ReadyReplicas     int32
	AuthoritativeAPI  string
}

// NewBackend creates appropriate backend instance based on configuration.
func NewBackend(backendType BackendType, authoritativeAPI BackendType) (MachineBackend, error) {
	switch backendType {
	case BackendTypeMAPI:
		return NewMAPIBackend(authoritativeAPI), nil
	case BackendTypeCAPI:
		return NewCAPIBackend(authoritativeAPI), nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

// NewMAPIBackend creates a new MAPI backend instance.
func NewMAPIBackend(authoritativeAPI BackendType) MachineBackend {
	return &mapiBackend{backendType: BackendTypeMAPI, authoritativeAPI: authoritativeAPI}
}

// NewCAPIBackend creates a new CAPI backend instance.
func NewCAPIBackend(authoritativeAPI BackendType) MachineBackend {
	return &capiBackend{backendType: BackendTypeCAPI, authoritativeAPI: authoritativeAPI}
}
