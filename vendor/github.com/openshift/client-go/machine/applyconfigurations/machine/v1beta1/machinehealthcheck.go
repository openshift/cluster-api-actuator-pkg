// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta1

import (
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	internal "github.com/openshift/client-go/machine/applyconfigurations/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	managedfields "k8s.io/apimachinery/pkg/util/managedfields"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// MachineHealthCheckApplyConfiguration represents a declarative configuration of the MachineHealthCheck type for use
// with apply.
type MachineHealthCheckApplyConfiguration struct {
	v1.TypeMetaApplyConfiguration    `json:",inline"`
	*v1.ObjectMetaApplyConfiguration `json:"metadata,omitempty"`
	Spec                             *MachineHealthCheckSpecApplyConfiguration   `json:"spec,omitempty"`
	Status                           *MachineHealthCheckStatusApplyConfiguration `json:"status,omitempty"`
}

// MachineHealthCheck constructs a declarative configuration of the MachineHealthCheck type for use with
// apply.
func MachineHealthCheck(name, namespace string) *MachineHealthCheckApplyConfiguration {
	b := &MachineHealthCheckApplyConfiguration{}
	b.WithName(name)
	b.WithNamespace(namespace)
	b.WithKind("MachineHealthCheck")
	b.WithAPIVersion("machine.openshift.io/v1beta1")
	return b
}

// ExtractMachineHealthCheck extracts the applied configuration owned by fieldManager from
// machineHealthCheck. If no managedFields are found in machineHealthCheck for fieldManager, a
// MachineHealthCheckApplyConfiguration is returned with only the Name, Namespace (if applicable),
// APIVersion and Kind populated. It is possible that no managed fields were found for because other
// field managers have taken ownership of all the fields previously owned by fieldManager, or because
// the fieldManager never owned fields any fields.
// machineHealthCheck must be a unmodified MachineHealthCheck API object that was retrieved from the Kubernetes API.
// ExtractMachineHealthCheck provides a way to perform a extract/modify-in-place/apply workflow.
// Note that an extracted apply configuration will contain fewer fields than what the fieldManager previously
// applied if another fieldManager has updated or force applied any of the previously applied fields.
// Experimental!
func ExtractMachineHealthCheck(machineHealthCheck *machinev1beta1.MachineHealthCheck, fieldManager string) (*MachineHealthCheckApplyConfiguration, error) {
	return extractMachineHealthCheck(machineHealthCheck, fieldManager, "")
}

// ExtractMachineHealthCheckStatus is the same as ExtractMachineHealthCheck except
// that it extracts the status subresource applied configuration.
// Experimental!
func ExtractMachineHealthCheckStatus(machineHealthCheck *machinev1beta1.MachineHealthCheck, fieldManager string) (*MachineHealthCheckApplyConfiguration, error) {
	return extractMachineHealthCheck(machineHealthCheck, fieldManager, "status")
}

func extractMachineHealthCheck(machineHealthCheck *machinev1beta1.MachineHealthCheck, fieldManager string, subresource string) (*MachineHealthCheckApplyConfiguration, error) {
	b := &MachineHealthCheckApplyConfiguration{}
	err := managedfields.ExtractInto(machineHealthCheck, internal.Parser().Type("com.github.openshift.api.machine.v1beta1.MachineHealthCheck"), fieldManager, b, subresource)
	if err != nil {
		return nil, err
	}
	b.WithName(machineHealthCheck.Name)
	b.WithNamespace(machineHealthCheck.Namespace)

	b.WithKind("MachineHealthCheck")
	b.WithAPIVersion("machine.openshift.io/v1beta1")
	return b, nil
}

// WithKind sets the Kind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kind field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithKind(value string) *MachineHealthCheckApplyConfiguration {
	b.TypeMetaApplyConfiguration.Kind = &value
	return b
}

// WithAPIVersion sets the APIVersion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIVersion field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithAPIVersion(value string) *MachineHealthCheckApplyConfiguration {
	b.TypeMetaApplyConfiguration.APIVersion = &value
	return b
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithName(value string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.Name = &value
	return b
}

// WithGenerateName sets the GenerateName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the GenerateName field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithGenerateName(value string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.GenerateName = &value
	return b
}

// WithNamespace sets the Namespace field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Namespace field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithNamespace(value string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.Namespace = &value
	return b
}

// WithUID sets the UID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UID field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithUID(value types.UID) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.UID = &value
	return b
}

// WithResourceVersion sets the ResourceVersion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ResourceVersion field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithResourceVersion(value string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.ResourceVersion = &value
	return b
}

// WithGeneration sets the Generation field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Generation field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithGeneration(value int64) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.Generation = &value
	return b
}

// WithCreationTimestamp sets the CreationTimestamp field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CreationTimestamp field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithCreationTimestamp(value metav1.Time) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.CreationTimestamp = &value
	return b
}

// WithDeletionTimestamp sets the DeletionTimestamp field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeletionTimestamp field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithDeletionTimestamp(value metav1.Time) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.DeletionTimestamp = &value
	return b
}

// WithDeletionGracePeriodSeconds sets the DeletionGracePeriodSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeletionGracePeriodSeconds field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithDeletionGracePeriodSeconds(value int64) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	b.ObjectMetaApplyConfiguration.DeletionGracePeriodSeconds = &value
	return b
}

// WithLabels puts the entries into the Labels field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Labels field,
// overwriting an existing map entries in Labels field with the same key.
func (b *MachineHealthCheckApplyConfiguration) WithLabels(entries map[string]string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	if b.ObjectMetaApplyConfiguration.Labels == nil && len(entries) > 0 {
		b.ObjectMetaApplyConfiguration.Labels = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.ObjectMetaApplyConfiguration.Labels[k] = v
	}
	return b
}

// WithAnnotations puts the entries into the Annotations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the Annotations field,
// overwriting an existing map entries in Annotations field with the same key.
func (b *MachineHealthCheckApplyConfiguration) WithAnnotations(entries map[string]string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	if b.ObjectMetaApplyConfiguration.Annotations == nil && len(entries) > 0 {
		b.ObjectMetaApplyConfiguration.Annotations = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		b.ObjectMetaApplyConfiguration.Annotations[k] = v
	}
	return b
}

// WithOwnerReferences adds the given value to the OwnerReferences field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the OwnerReferences field.
func (b *MachineHealthCheckApplyConfiguration) WithOwnerReferences(values ...*v1.OwnerReferenceApplyConfiguration) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithOwnerReferences")
		}
		b.ObjectMetaApplyConfiguration.OwnerReferences = append(b.ObjectMetaApplyConfiguration.OwnerReferences, *values[i])
	}
	return b
}

// WithFinalizers adds the given value to the Finalizers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Finalizers field.
func (b *MachineHealthCheckApplyConfiguration) WithFinalizers(values ...string) *MachineHealthCheckApplyConfiguration {
	b.ensureObjectMetaApplyConfigurationExists()
	for i := range values {
		b.ObjectMetaApplyConfiguration.Finalizers = append(b.ObjectMetaApplyConfiguration.Finalizers, values[i])
	}
	return b
}

func (b *MachineHealthCheckApplyConfiguration) ensureObjectMetaApplyConfigurationExists() {
	if b.ObjectMetaApplyConfiguration == nil {
		b.ObjectMetaApplyConfiguration = &v1.ObjectMetaApplyConfiguration{}
	}
}

// WithSpec sets the Spec field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Spec field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithSpec(value *MachineHealthCheckSpecApplyConfiguration) *MachineHealthCheckApplyConfiguration {
	b.Spec = value
	return b
}

// WithStatus sets the Status field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Status field is set to the value of the last call.
func (b *MachineHealthCheckApplyConfiguration) WithStatus(value *MachineHealthCheckStatusApplyConfiguration) *MachineHealthCheckApplyConfiguration {
	b.Status = value
	return b
}

// GetName retrieves the value of the Name field in the declarative configuration.
func (b *MachineHealthCheckApplyConfiguration) GetName() *string {
	b.ensureObjectMetaApplyConfigurationExists()
	return b.ObjectMetaApplyConfiguration.Name
}
