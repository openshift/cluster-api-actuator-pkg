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

	corev1 "k8s.io/api/core/v1"
	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackMachineBuilder", func() {
	Describe("Build", func() {
		It("should return a default OpenStackMachine when no options are specified", func() {
			openstackMachine := OpenStackMachine().Build()
			Expect(openstackMachine).ToNot(BeNil())
			Expect(openstackMachine.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(openstackMachine.TypeMeta.Kind).To(Equal("OpenStackMachine"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			openstackMachine := OpenStackMachine().WithAnnotations(annotations).Build()
			Expect(openstackMachine.Annotations).To(Equal(annotations))
		})
	})

	// Spec fields.

	Describe("WithAdditionalBlockDevices", func() {
		It("should return the custom value when specified", func() {
			bdms := []capov1.AdditionalBlockDevice{{Name: "my-volume"}}
			openstackMachine := OpenStackMachine().WithAdditionalBlockDevices(bdms).Build()
			Expect(openstackMachine.Spec.AdditionalBlockDevices).To(Equal(bdms))
		})
	})

	// Status fields.

	Describe("WithAddresses", func() {
		It("should return the custom value when specified", func() {
			addresses := []corev1.NodeAddress{{Type: corev1.NodeExternalIP, Address: "192.168.1.1"}}
			openstackMachine := OpenStackMachine().WithAddresses(addresses).Build()
			Expect(openstackMachine.Status.Addresses).To(Equal(addresses))
		})
	})
})
