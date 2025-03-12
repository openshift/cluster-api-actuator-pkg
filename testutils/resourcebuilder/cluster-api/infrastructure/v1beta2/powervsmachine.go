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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// PowerVSMachine creates a new PowerVSMachine builder.
func PowerVSMachine() PowerVSMachineBuilder {
	return PowerVSMachineBuilder{}
}

// PowerVSMachineBuilder is used to build out an PowerVSMachine object.
type PowerVSMachineBuilder struct {
	// ObjectMeta fields.
	annotations map[string]string
	labels      map[string]string
	name        string
	namespace   string

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
	addresses      []corev1.NodeAddress
	conditions     clusterv1.Conditions
	failureMessage *string
	failureReason  *string
	instanceID     string
	instanceState  capibmv1.PowerVSInstanceState
	ready          bool
}

func (p PowerVSMachineBuilder) Build() *capibmv1.IBMPowerVSMachine {
	return &capibmv1.IBMPowerVSMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IBMPowerVSMachine",
			APIVersion: capibmv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.name,
			Namespace:   p.namespace,
			Labels:      p.labels,
			Annotations: p.annotations,
		},
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
		Status: capibmv1.IBMPowerVSMachineStatus{
			InstanceID:     p.instanceID,
			Ready:          p.ready,
			Addresses:      p.addresses,
			InstanceState:  p.instanceState,
			FailureMessage: p.failureMessage,
			FailureReason:  p.failureReason,
			Conditions:     p.conditions,
		},
	}
}

// Object meta fields.

// WithAnnotations sets the annotations for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithAnnotations(annotations map[string]string) PowerVSMachineBuilder {
	p.annotations = annotations
	return p
}

// WithLabels sets the labels for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithLabels(labels map[string]string) PowerVSMachineBuilder {
	p.labels = labels
	return p
}

// WithName sets the name for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithName(name string) PowerVSMachineBuilder {
	p.name = name
	return p
}

// WithNamespace sets the namespace for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithNamespace(namespace string) PowerVSMachineBuilder {
	p.namespace = namespace
	return p
}

// Spec fields.

// WithImage sets the image for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithImage(image *capibmv1.IBMPowerVSResourceReference) PowerVSMachineBuilder {
	p.image = image
	return p
}

// WithImageRef sets the imageRef for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithImageRef(imageRef *corev1.LocalObjectReference) PowerVSMachineBuilder {
	p.imageRef = imageRef
	return p
}

// WithMemoryGiB sets the memoryGiB for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithMemoryGiB(memoryGiB int32) PowerVSMachineBuilder {
	p.memoryGiB = memoryGiB
	return p
}

// WithNetwork sets the network for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithNetwork(network capibmv1.IBMPowerVSResourceReference) PowerVSMachineBuilder {
	p.network = network
	return p
}

// WithProcessors sets the processors for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithProcessors(processors intstr.IntOrString) PowerVSMachineBuilder {
	p.processors = processors
	return p
}

// WithProcessorType sets the processorType for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithProcessorType(processorType capibmv1.PowerVSProcessorType) PowerVSMachineBuilder {
	p.processorType = processorType
	return p
}

// WithProviderID sets the providerID for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithProviderID(providerID *string) PowerVSMachineBuilder {
	p.providerID = providerID
	return p
}

// WithServiceInstance sets the serviceInstance for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithServiceInstance(serviceInstance *capibmv1.IBMPowerVSResourceReference) PowerVSMachineBuilder {
	p.serviceInstance = serviceInstance
	return p
}

// WithSSHKey sets the sshKey for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithSSHKey(sshKey string) PowerVSMachineBuilder {
	p.sshKey = sshKey
	return p
}

// WithSystemType sets the systemType for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithSystemType(systemType string) PowerVSMachineBuilder {
	p.systemType = systemType
	return p
}

// Status fields.

// WithAddresses sets the addresses for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithAddresses(addresses []corev1.NodeAddress) PowerVSMachineBuilder {
	p.addresses = addresses
	return p
}

// WithConditions sets the conditions for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithConditions(conditions clusterv1.Conditions) PowerVSMachineBuilder {
	p.conditions = conditions
	return p
}

// WithFailureMessage sets the failureMessage for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithFailureMessage(failureMessage *string) PowerVSMachineBuilder {
	p.failureMessage = failureMessage
	return p
}

// WithFailureReason sets the failureReason for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithFailureReason(failureReason *string) PowerVSMachineBuilder {
	p.failureReason = failureReason
	return p
}

// WithInstanceID sets the instanceID for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithInstanceID(instanceID string) PowerVSMachineBuilder {
	p.instanceID = instanceID
	return p
}

// WithInstanceState sets the instanceState for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithInstanceState(instanceState capibmv1.PowerVSInstanceState) PowerVSMachineBuilder {
	p.instanceState = instanceState
	return p
}

// WithReady sets the ready for the PowerVSMachine builder.
func (p PowerVSMachineBuilder) WithReady(ready bool) PowerVSMachineBuilder {
	p.ready = ready
	return p
}
