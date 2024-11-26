/*
Copyright 2024 Red Hat, Inc.

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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	capibmv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
)

// PowerVSMachineTemplate creates a new PowerVSMachineTemplate builder.
func PowerVSMachineTemplate() PowerVSMachineTemplateBuilder {
	return PowerVSMachineTemplateBuilder{}
}

// PowerVSMachineTemplateBuilder is used to build out an PowerVSMachineTemplate object.
type PowerVSMachineTemplateBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string

	// Spec fields.
	image           *capibmv1.IBMPowerVSResourceReference
	imageRef        *corev1.LocalObjectReference
	memoryGiB       int32
	network         capibmv1.IBMPowerVSResourceReference
	processors      intstr.IntOrString
	processorType   capibmv1.PowerVSProcessorType
	providerID      *string
	serviceInstance *capibmv1.IBMPowerVSResourceReference
	sshKey          string
	systemType      string

	// Status fields.
	capacity corev1.ResourceList
}

// Build builds a new PowerVSMachineTemplate based on the configuration provided.
func (p PowerVSMachineTemplateBuilder) Build() *capibmv1.IBMPowerVSMachineTemplate {
	return &capibmv1.IBMPowerVSMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IBMPowerVSMachineTemplate",
			APIVersion: capibmv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       p.annotations,
			CreationTimestamp: p.creationTimestamp,
			DeletionTimestamp: p.deletionTimestamp,
			GenerateName:      p.generateName,
			Labels:            p.labels,
			Name:              p.name,
			Namespace:         p.namespace,
		},
		Spec: capibmv1.IBMPowerVSMachineTemplateSpec{
			Template: capibmv1.IBMPowerVSMachineTemplateResource{
				Spec: capibmv1.IBMPowerVSMachineSpec{
					ServiceInstance: p.serviceInstance,
					SSHKey:          p.sshKey,
					Image:           p.image,
					ImageRef:        p.imageRef,
					SystemType:      p.systemType,
					ProcessorType:   p.processorType,
					Processors:      p.processors,
					MemoryGiB:       p.memoryGiB,
					Network:         p.network,
					ProviderID:      p.providerID,
				},
			},
		},
		Status: capibmv1.IBMPowerVSMachineTemplateStatus{
			Capacity: p.capacity,
		},
	}
}

// Object meta fields.

// WithAnnotations sets the annotations for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithAnnotations(annotations map[string]string) PowerVSMachineTemplateBuilder {
	p.annotations = annotations
	return p
}

// WithCreationTimestamp sets the creationTimestamp for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithCreationTimestamp(timestamp metav1.Time) PowerVSMachineTemplateBuilder {
	p.creationTimestamp = timestamp
	return p
}

// WithDeletionTimestamp sets the deletionTimestamp for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithDeletionTimestamp(timestamp *metav1.Time) PowerVSMachineTemplateBuilder {
	p.deletionTimestamp = timestamp
	return p
}

// WithGenerateName sets the generateName for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithGenerateName(generateName string) PowerVSMachineTemplateBuilder {
	p.generateName = generateName
	return p
}

// WithLabels sets the labels for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithLabels(labels map[string]string) PowerVSMachineTemplateBuilder {
	p.labels = labels
	return p
}

// WithName sets the name for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithName(name string) PowerVSMachineTemplateBuilder {
	p.name = name
	return p
}

// WithNamespace sets the namespace for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithNamespace(namespace string) PowerVSMachineTemplateBuilder {
	p.namespace = namespace
	return p
}

// Spec fields.

// WithImage sets the image for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithImage(image *capibmv1.IBMPowerVSResourceReference) PowerVSMachineTemplateBuilder {
	p.image = image
	return p
}

// WithImageRef sets the imageRef for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithImageRef(imageRef *corev1.LocalObjectReference) PowerVSMachineTemplateBuilder {
	p.imageRef = imageRef
	return p
}

// WithMemoryGiB sets the memoryGiB for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithMemoryGiB(memoryGiB int32) PowerVSMachineTemplateBuilder {
	p.memoryGiB = memoryGiB
	return p
}

// WithNetwork sets the network for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithNetwork(network capibmv1.IBMPowerVSResourceReference) PowerVSMachineTemplateBuilder {
	p.network = network
	return p
}

// WithProcessors sets the processors for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithProcessors(processors intstr.IntOrString) PowerVSMachineTemplateBuilder {
	p.processors = processors
	return p
}

// WithProcessorType sets the processorType for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithProcessorType(processorType capibmv1.PowerVSProcessorType) PowerVSMachineTemplateBuilder {
	p.processorType = processorType
	return p
}

// WithProviderID sets the providerID for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithProviderID(providerID *string) PowerVSMachineTemplateBuilder {
	p.providerID = providerID
	return p
}

// WithServiceInstance sets the serviceInstance for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithServiceInstance(serviceInstance *capibmv1.IBMPowerVSResourceReference) PowerVSMachineTemplateBuilder {
	p.serviceInstance = serviceInstance
	return p
}

// WithSSHKey sets the sshKey for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithSSHKey(sshKey string) PowerVSMachineTemplateBuilder {
	p.sshKey = sshKey
	return p
}

// WithSystemType sets the systemType for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithSystemType(systemType string) PowerVSMachineTemplateBuilder {
	p.systemType = systemType
	return p
}

// Status fields.

// WithCapacity sets the capacity for the PowerVSMachineTemplate builder.
func (p PowerVSMachineTemplateBuilder) WithCapacity(capacity corev1.ResourceList) PowerVSMachineTemplateBuilder {
	p.capacity = capacity
	return p
}
