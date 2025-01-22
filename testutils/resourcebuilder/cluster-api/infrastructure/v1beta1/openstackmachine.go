/*
Copyright 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// OpenStackMachine creates a new OpenStackMachine builder.
func OpenStackMachine() OpenStackMachineBuilder {
	return OpenStackMachineBuilder{}
}

// OpenStackMachineBuilder is used to build out an OpenStackMachine object.
type OpenStackMachineBuilder struct {
	// ObjectMeta fields.
	annotations map[string]string
	labels      map[string]string
	name        string
	namespace   string

	// Spec fields.
	additionalBlockDevices            []capov1.AdditionalBlockDevice
	configDrive                       *bool
	flavor                            *string
	flavorID                          *string
	floatingIPPoolRef                 *corev1.TypedLocalObjectReference
	identityRef                       *capov1.OpenStackIdentityReference
	image                             capov1.ImageParam
	ports                             []capov1.PortOpts
	rootVolume                        *capov1.RootVolume
	schedulerHintAdditionalProperties []capov1.SchedulerHintAdditionalProperty
	securityGroups                    []capov1.SecurityGroupParam
	serverGroup                       *capov1.ServerGroupParam
	serverMetadata                    []capov1.ServerMetadata
	sshKeyName                        string
	tags                              []string
	trunk                             bool

	// Status fields.
	addresses      []corev1.NodeAddress
	conditions     clusterv1.Conditions
	failureMessage *string
	failureReason  *errors.MachineStatusError
	instanceID     *string
	instanceState  *capov1.InstanceState
	ready          bool
}

// Build builds a new OpenStackMachine based on the configuration provided.
func (a OpenStackMachineBuilder) Build() *capov1.OpenStackMachine {
	openstackMachine := &capov1.OpenStackMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capov1.SchemeGroupVersion.String(),
			Kind:       "OpenStackMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.name,
			Namespace:   a.namespace,
			Labels:      a.labels,
			Annotations: a.annotations,
		},
		Spec: capov1.OpenStackMachineSpec{
			AdditionalBlockDevices:            a.additionalBlockDevices,
			ConfigDrive:                       a.configDrive,
			Flavor:                            a.flavor,
			FlavorID:                          a.flavorID,
			FloatingIPPoolRef:                 a.floatingIPPoolRef,
			IdentityRef:                       a.identityRef,
			Image:                             a.image,
			Ports:                             a.ports,
			RootVolume:                        a.rootVolume,
			SchedulerHintAdditionalProperties: a.schedulerHintAdditionalProperties,
			SecurityGroups:                    a.securityGroups,
			ServerGroup:                       a.serverGroup,
			ServerMetadata:                    a.serverMetadata,
			SSHKeyName:                        a.sshKeyName,
			Tags:                              a.tags,
			Trunk:                             a.trunk,
		},
		Status: capov1.OpenStackMachineStatus{
			Addresses:      a.addresses,
			Conditions:     a.conditions,
			FailureMessage: a.failureMessage,
			FailureReason:  a.failureReason,
			InstanceID:     a.instanceID,
			InstanceState:  a.instanceState,
			Ready:          a.ready,
		},
	}

	return openstackMachine
}

// Object meta fields.

// WithAnnotations sets the annotations for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithAnnotations(annotations map[string]string) OpenStackMachineBuilder {
	a.annotations = annotations
	return a
}

// WithLabels sets the labels for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithLabels(labels map[string]string) OpenStackMachineBuilder {
	a.labels = labels
	return a
}

// WithName sets the name for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithName(name string) OpenStackMachineBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithNamespace(namespace string) OpenStackMachineBuilder {
	a.namespace = namespace
	return a
}

// Spec fields.

// WithAdditionalBlockDevices sets the additionalBlockDevices for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithAdditionalBlockDevices(additionalBlockDevices []capov1.AdditionalBlockDevice) OpenStackMachineBuilder {
	a.additionalBlockDevices = additionalBlockDevices
	return a
}

// WithConfigDrive sets the configDrive for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithConfigDrive(configDrive *bool) OpenStackMachineBuilder {
	a.configDrive = configDrive
	return a
}

// WithFlavor sets the flavor for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithFlavor(flavor *string) OpenStackMachineBuilder {
	a.flavor = flavor
	return a
}

// WithFlavorID sets the flavorID for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithFlavorID(flavorID *string) OpenStackMachineBuilder {
	a.flavorID = flavorID
	return a
}

// WithIdentityRef sets the identityRef for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithIdentityRef(identityRef *capov1.OpenStackIdentityReference) OpenStackMachineBuilder {
	a.identityRef = identityRef
	return a
}

// WithImage sets the image for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithImage(image capov1.ImageParam) OpenStackMachineBuilder {
	a.image = image
	return a
}

// WithPorts sets the ports for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithPorts(ports []capov1.PortOpts) OpenStackMachineBuilder {
	a.ports = ports
	return a
}

// WithRootVolume sets the rootVolume for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithRootVolume(rootVolume *capov1.RootVolume) OpenStackMachineBuilder {
	a.rootVolume = rootVolume
	return a
}

// WithSchedulerHintAdditionalProperties sets the schedulerHintAdditionalProperties for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithSchedulerHintAdditionalProperties(schedulerHintAdditionalProperties []capov1.SchedulerHintAdditionalProperty) OpenStackMachineBuilder {
	a.schedulerHintAdditionalProperties = schedulerHintAdditionalProperties
	return a
}

// WithSecurityGroups sets the securityGroups for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithSecurityGroups(securityGroups []capov1.SecurityGroupParam) OpenStackMachineBuilder {
	a.securityGroups = securityGroups
	return a
}

// WithServerGroup sets the serverGroup for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithServerGroup(serverGroup *capov1.ServerGroupParam) OpenStackMachineBuilder {
	a.serverGroup = serverGroup
	return a
}

// WithServerMetadata sets the serverMetadata for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithServerMetadata(serverMetadata []capov1.ServerMetadata) OpenStackMachineBuilder {
	a.serverMetadata = serverMetadata
	return a
}

// WithSSHKeyName sets the sshKeyName for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithSSHKeyName(sshKeyName string) OpenStackMachineBuilder {
	a.sshKeyName = sshKeyName
	return a
}

// WithTags sets the tags for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithTags(tags []string) OpenStackMachineBuilder {
	a.tags = tags
	return a
}

// WithTrunk sets the trunk for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithTrunk(trunk bool) OpenStackMachineBuilder {
	a.trunk = trunk
	return a
}

// Status fields.

// WithAddresses sets the addresses for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithAddresses(addresses []corev1.NodeAddress) OpenStackMachineBuilder {
	a.addresses = addresses
	return a
}

// WithConditions sets the conditions for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithConditions(conditions clusterv1.Conditions) OpenStackMachineBuilder {
	a.conditions = conditions
	return a
}

// WithFailureMessage sets the failureMessage for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithFailureMessage(failureMessage *string) OpenStackMachineBuilder {
	a.failureMessage = failureMessage
	return a
}

// WithFailureReason sets the failureReason for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithFailureReason(failureReason *errors.MachineStatusError) OpenStackMachineBuilder {
	a.failureReason = failureReason
	return a
}

// WithInstanceID sets the instanceID for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithInstanceID(instanceID *string) OpenStackMachineBuilder {
	a.instanceID = instanceID
	return a
}

// WithInstanceState sets the instanceState for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithInstanceState(instanceState *capov1.InstanceState) OpenStackMachineBuilder {
	a.instanceState = instanceState
	return a
}

// WithReady sets the ready for the OpenStackMachine builder.
func (a OpenStackMachineBuilder) WithReady(ready bool) OpenStackMachineBuilder {
	a.ready = ready
	return a
}
