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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	capibmv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
)

var _ = Describe("PowerVSMachineTemplate", func() {
	Describe("Build", func() {
		It("should return a default PowerVSMachineTemplate when no options are specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().Build()
			Expect(powerVSMachineTemplate).ToNot(BeNil())
			Expect(powerVSMachineTemplate.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(powerVSMachineTemplate.TypeMeta.Kind).To(Equal("IBMPowerVSMachineTemplate"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			powerVSMachineTemplate := PowerVSMachineTemplate().WithAnnotations(annotations).Build()
			Expect(powerVSMachineTemplate.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			powerVSMachineTemplate := PowerVSMachineTemplate().WithLabels(labels).Build()
			Expect(powerVSMachineTemplate.Labels).To(Equal(labels))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-"
			powerVSMachineTemplate := PowerVSMachineTemplate().WithGenerateName(generateName).Build()
			Expect(powerVSMachineTemplate.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			powerVSMachineTemplate := PowerVSMachineTemplate().WithCreationTimestamp(timestamp).Build()
			Expect(powerVSMachineTemplate.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			powerVSMachineTemplate := PowerVSMachineTemplate().WithDeletionTimestamp(&timestamp).Build()
			Expect(powerVSMachineTemplate.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-aws-machine-template"
			powerVSMachineTemplate := PowerVSMachineTemplate().WithName(name).Build()
			Expect(powerVSMachineTemplate.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithNamespace("test-ns").Build()
			Expect(powerVSMachineTemplate.Namespace).To(Equal("test-ns"))
		})
	})

	// Spec fields.

	Describe("WithServiceInstance", func() {
		serviceInstance := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("service-instance")}
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithServiceInstance(serviceInstance).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.ServiceInstance).To(Equal(serviceInstance))
		})
	})

	Describe("WithSSHKey", func() {
		sshKey := "ssh-key"
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithSSHKey(sshKey).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.SSHKey).To(Equal(sshKey))
		})
	})

	Describe("WithImage", func() {
		image := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("image")}
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithImage(image).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.Image).To(Equal(image))
		})
	})

	Describe("WithImageRef", func() {
		imageRef := &corev1.LocalObjectReference{Name: "image"}
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithImageRef(imageRef).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.ImageRef).To(Equal(imageRef))
		})
	})

	Describe("WithSystemType", func() {
		systemType := "systemType"
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithSystemType(systemType).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.SystemType).To(Equal(systemType))
		})
	})

	Describe("WithProcessorType", func() {
		processorType := capibmv1.PowerVSProcessorTypeShared
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithProcessorType(processorType).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.ProcessorType).To(Equal(processorType))
		})
	})

	Describe("WithProcessors", func() {
		processors := intstr.FromString("2")
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithProcessors(processors).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.Processors).To(Equal(processors))
		})
	})

	Describe("WithMemoryGiB", func() {
		var memory int32 = 3
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithMemoryGiB(memory).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.MemoryGiB).To(Equal(memory))
		})
	})

	Describe("WithNetwork", func() {
		network := capibmv1.IBMPowerVSResourceReference{Name: ptr.To("network-name")}
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithNetwork(network).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.Network).To(Equal(network))
		})
	})

	Describe("WithProviderID", func() {
		providerID := ptr.To("provider-id")
		It("should return the custom value when specified", func() {
			powerVSMachineTemplate := PowerVSMachineTemplate().WithProviderID(providerID).Build()
			Expect(powerVSMachineTemplate.Spec.Template.Spec.ProviderID).To(Equal(providerID))
		})
	})

	// Status fields.

	Describe("WithCapacity", func() {
		It("should return the custom value when specified", func() {
			capacity := corev1.ResourceList{"cpu": resource.MustParse("2")}
			powerVSMachineTemplate := PowerVSMachineTemplate().WithCapacity(capacity).Build()
			Expect(powerVSMachineTemplate.Status.Capacity).To(Equal(capacity))
		})
	})
})
