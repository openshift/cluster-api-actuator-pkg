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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	capibmv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("PowerVSMachine", func() {
	Describe("Build", func() {
		It("should return a default PowerVSMachine when no options are specified", func() {
			powerVSMachine := PowerVSMachine().Build()
			Expect(powerVSMachine).ToNot(BeNil())
			Expect(powerVSMachine.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(powerVSMachine.TypeMeta.Kind).To(Equal("IBMPowerVSMachine"))
		})
	})

	// Object meta fields

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			PowerVSMachine := PowerVSMachine().WithAnnotations(annotations).Build()
			Expect(PowerVSMachine.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			PowerVSMachine := PowerVSMachine().WithLabels(labels).Build()
			Expect(PowerVSMachine.Labels).To(Equal(labels))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-aws-machine"
			PowerVSMachine := PowerVSMachine().WithName(name).Build()
			Expect(PowerVSMachine.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			PowerVSMachine := PowerVSMachine().WithNamespace("ns-test-5").Build()
			Expect(PowerVSMachine.Namespace).To(Equal("ns-test-5"))
		})
	})

	// Spec fields.

	Describe("WithImage", func() {
		image := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("image")}
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithImage(image).Build()
			Expect(powerVSMachine.Spec.Image).To(Equal(image))
		})
	})

	Describe("WithImageRef", func() {
		imageRef := &corev1.LocalObjectReference{Name: "image"}
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithImageRef(imageRef).Build()
			Expect(powerVSMachine.Spec.ImageRef).To(Equal(imageRef))
		})
	})

	Describe("WithMemoryGiB", func() {
		var memory int32 = 3
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithMemoryGiB(memory).Build()
			Expect(powerVSMachine.Spec.MemoryGiB).To(Equal(memory))
		})
	})

	Describe("WithNetwork", func() {
		network := capibmv1.IBMPowerVSResourceReference{Name: ptr.To("network-name")}
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithNetwork(network).Build()
			Expect(powerVSMachine.Spec.Network).To(Equal(network))
		})
	})

	Describe("WithProcessors", func() {
		processors := intstr.FromString("2")
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithProcessors(processors).Build()
			Expect(powerVSMachine.Spec.Processors).To(Equal(processors))
		})
	})

	Describe("WithProcessorType", func() {
		processorType := capibmv1.PowerVSProcessorTypeShared
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithProcessorType(processorType).Build()
			Expect(powerVSMachine.Spec.ProcessorType).To(Equal(processorType))
		})
	})

	Describe("WithProviderID", func() {
		providerID := ptr.To("provider-id")
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithProviderID(providerID).Build()
			Expect(powerVSMachine.Spec.ProviderID).To(Equal(providerID))
		})
	})

	Describe("WithServiceInstance", func() {
		serviceInstance := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("service-instance")}
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithServiceInstance(serviceInstance).Build()
			Expect(powerVSMachine.Spec.ServiceInstance).To(Equal(serviceInstance))
		})
	})

	Describe("WithSSHKey", func() {
		sshKey := "ssh-key"
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithSSHKey(sshKey).Build()
			Expect(powerVSMachine.Spec.SSHKey).To(Equal(sshKey))
		})
	})

	Describe("WithSystemType", func() {
		systemType := "systemType"
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithSystemType(systemType).Build()
			Expect(powerVSMachine.Spec.SystemType).To(Equal(systemType))
		})
	})

	// Status fields.

	Describe("WithAddresses", func() {
		It("should return the custom value when specified", func() {
			addresses := []corev1.NodeAddress{{Type: corev1.NodeExternalIP, Address: "192.168.1.1"}}
			powerVSMachine := PowerVSMachine().WithAddresses(addresses).Build()
			Expect(powerVSMachine.Status.Addresses).To(Equal(addresses))
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := clusterv1.Conditions{{Type: clusterv1.ReadyCondition, Status: corev1.ConditionTrue}}
			powerVSMachine := PowerVSMachine().WithConditions(conditions).Build()
			Expect(powerVSMachine.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithFailureMessage", func() {
		It("should return the custom value when specified", func() {
			message := "test failure message"
			powerVSMachine := PowerVSMachine().WithFailureMessage(&message).Build()
			Expect(powerVSMachine.Status.FailureMessage).To(Equal(&message))
		})
	})

	Describe("WithFailureReason", func() {
		It("should return the custom value when specified", func() {
			reason := "CreateError"
			powerVSMachine := PowerVSMachine().WithFailureReason(&reason).Build()
			Expect(powerVSMachine.Status.FailureReason).To(Equal(&reason))
		})
	})
	Describe("WithInstanceID", func() {
		instanceID := "instance-id"
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithInstanceID(instanceID).Build()
			Expect(powerVSMachine.Status.InstanceID).To(Equal(instanceID))
		})
	})

	Describe("WithInstanceState", func() {
		instanceState := capibmv1.PowerVSInstanceStateACTIVE
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithInstanceState(instanceState).Build()
			Expect(powerVSMachine.Status.InstanceState).To(Equal(instanceState))
		})
	})

	Describe("WithReady", func() {
		It("should return the custom value when specified", func() {
			powerVSMachine := PowerVSMachine().WithReady(true).Build()
			Expect(powerVSMachine.Status.Ready).To(Equal(true))
		})
	})
})
