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
)

// OpenStackMachineTemplate creates a new OpenStackMachineTemplate builder.
func OpenStackMachineTemplate() OpenStackMachineTemplateBuilder {
	return OpenStackMachineTemplateBuilder{}
}

// OpenStackMachineTemplateBuilder is used to build out an OpenStackMachineTemplate object.
type OpenStackMachineTemplateBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string

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
}

// Build builds a new OpenStackMachineTemplate based on the configuration provided.
func (a OpenStackMachineTemplateBuilder) Build() *capov1.OpenStackMachineTemplate {
	openstackMachineTemplate := &capov1.OpenStackMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capov1.SchemeGroupVersion.String(),
			Kind:       "OpenStackMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       a.annotations,
			CreationTimestamp: a.creationTimestamp,
			DeletionTimestamp: a.deletionTimestamp,
			GenerateName:      a.generateName,
			Labels:            a.labels,
			Name:              a.name,
			Namespace:         a.namespace,
		},
		Spec: capov1.OpenStackMachineTemplateSpec{
			Template: capov1.OpenStackMachineTemplateResource{
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
			},
		},
	}

	return openstackMachineTemplate
}

// Object meta fields.

// WithAnnotations sets the annotations for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithAnnotations(annotations map[string]string) OpenStackMachineTemplateBuilder {
	a.annotations = annotations
	return a
}

// WithCreationTimestamp sets the creationTimestamp for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithCreationTimestamp(timestamp metav1.Time) OpenStackMachineTemplateBuilder {
	a.creationTimestamp = timestamp
	return a
}

// WithDeletionTimestamp sets the deletionTimestamp for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithDeletionTimestamp(timestamp *metav1.Time) OpenStackMachineTemplateBuilder {
	a.deletionTimestamp = timestamp
	return a
}

// WithGenerateName sets the generateName for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithGenerateName(generateName string) OpenStackMachineTemplateBuilder {
	a.generateName = generateName
	return a
}

// WithLabels sets the labels for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithLabels(labels map[string]string) OpenStackMachineTemplateBuilder {
	a.labels = labels
	return a
}

// WithName sets the name for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithName(name string) OpenStackMachineTemplateBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the OpenStackMachineTemplate builder.
func (a OpenStackMachineTemplateBuilder) WithNamespace(namespace string) OpenStackMachineTemplateBuilder {
	a.namespace = namespace
	return a
}

// Spec fields.

// WithAdditionalBlockDevices sets the additionalBlockDevices for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithAdditionalBlockDevices(additionalBlockDevices []capov1.AdditionalBlockDevice) OpenStackMachineTemplateBuilder {
	a.additionalBlockDevices = additionalBlockDevices
	return a
}

// WithConfigDrive sets the configDrive for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithConfigDrive(configDrive *bool) OpenStackMachineTemplateBuilder {
	a.configDrive = configDrive
	return a
}

// WithFlavor sets the flavor for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithFlavor(flavor *string) OpenStackMachineTemplateBuilder {
	a.flavor = flavor
	return a
}

// WithFlavorID sets the flavorID for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithFlavorID(flavorID *string) OpenStackMachineTemplateBuilder {
	a.flavorID = flavorID
	return a
}

// WithIdentityRef sets the identityRef for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithIdentityRef(identityRef *capov1.OpenStackIdentityReference) OpenStackMachineTemplateBuilder {
	a.identityRef = identityRef
	return a
}

// WithImage sets the image for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithImage(image capov1.ImageParam) OpenStackMachineTemplateBuilder {
	a.image = image
	return a
}

// WithPorts sets the ports for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithPorts(ports []capov1.PortOpts) OpenStackMachineTemplateBuilder {
	a.ports = ports
	return a
}

// WithRootVolume sets the rootVolume for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithRootVolume(rootVolume *capov1.RootVolume) OpenStackMachineTemplateBuilder {
	a.rootVolume = rootVolume
	return a
}

// WithSchedulerHintAdditionalProperties sets the schedulerHintAdditionalProperties for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithSchedulerHintAdditionalProperties(schedulerHintAdditionalProperties []capov1.SchedulerHintAdditionalProperty) OpenStackMachineTemplateBuilder {
	a.schedulerHintAdditionalProperties = schedulerHintAdditionalProperties
	return a
}

// WithSecurityGroups sets the securityGroups for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithSecurityGroups(securityGroups []capov1.SecurityGroupParam) OpenStackMachineTemplateBuilder {
	a.securityGroups = securityGroups
	return a
}

// WithServerGroup sets the serverGroup for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithServerGroup(serverGroup *capov1.ServerGroupParam) OpenStackMachineTemplateBuilder {
	a.serverGroup = serverGroup
	return a
}

// WithServerMetadata sets the serverMetadata for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithServerMetadata(serverMetadata []capov1.ServerMetadata) OpenStackMachineTemplateBuilder {
	a.serverMetadata = serverMetadata
	return a
}

// WithSSHKeyName sets the sshKeyName for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithSSHKeyName(sshKeyName string) OpenStackMachineTemplateBuilder {
	a.sshKeyName = sshKeyName
	return a
}

// WithTags sets the tags for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithTags(tags []string) OpenStackMachineTemplateBuilder {
	a.tags = tags
	return a
}

// WithTrunk sets the trunk for the OpenStackMachine builder.
func (a OpenStackMachineTemplateBuilder) WithTrunk(trunk bool) OpenStackMachineTemplateBuilder {
	a.trunk = trunk
	return a
}
