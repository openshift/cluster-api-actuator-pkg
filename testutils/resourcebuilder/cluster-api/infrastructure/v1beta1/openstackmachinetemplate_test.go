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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackMachineTemplateBuilder", func() {
	Describe("Build", func() {
		It("should return a default OpenStackMachineTemplate when no options are specified", func() {
			openstackMachineTemplate := OpenStackMachineTemplate().Build()
			Expect(openstackMachineTemplate).ToNot(BeNil())
			Expect(openstackMachineTemplate.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(openstackMachineTemplate.TypeMeta.Kind).To(Equal("OpenStackMachineTemplate"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			openstackMachineTemplate := OpenStackMachineTemplate().WithAnnotations(annotations).Build()
			Expect(openstackMachineTemplate.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerReferences := []metav1.OwnerReference{{Name: "cluster"}}
			openstackMachineTemplate := OpenStackMachineTemplate().WithOwnerReferences(ownerReferences).Build()
			Expect(openstackMachineTemplate.OwnerReferences).To(Equal(ownerReferences))
		})
	})

	// Spec fields.

	Describe("WithAdditionalBlockDevices", func() {
		It("should return the custom value when specified", func() {
			bdms := []capov1.AdditionalBlockDevice{{Name: "my-volume"}}
			openstackMachineTemplate := OpenStackMachineTemplate().WithAdditionalBlockDevices(bdms).Build()
			Expect(openstackMachineTemplate.Spec.Template.Spec.AdditionalBlockDevices).To(Equal(bdms))
		})
	})
})
