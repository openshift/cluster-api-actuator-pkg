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

	"k8s.io/utils/ptr"
	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
)

var _ = Describe("OpenStackCluster", func() {
	Describe("Build", func() {
		It("should return a default OpenStackCluster when no options are specified", func() {
			openstackCluster := OpenStackCluster().Build()
			Expect(openstackCluster).ToNot(BeNil())
			Expect(openstackCluster.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(openstackCluster.TypeMeta.Kind).To(Equal("OpenStackCluster"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			openstackCluster := OpenStackCluster().WithAnnotations(annotations).Build()
			Expect(openstackCluster.Annotations).To(Equal(annotations))
		})
	})

	// Spec fields.

	Describe("WithAPIServerFixedIP", func() {
		It("should return the custom value when specified", func() {
			fixedIP := ptr.To("192.168.25.10")
			openstackCluster := OpenStackCluster().WithAPIServerFixedIP(fixedIP).Build()
			Expect(openstackCluster.Spec.APIServerFixedIP).To(Equal(fixedIP))
		})
	})

	// Status fields.

	Describe("WithBastionStatus", func() {
		It("should return the custom value when specified", func() {
			bastionStatus := &capov1.BastionStatus{ID: "e5f511f7-961c-4a85-a5c9-6443844a7ada"}
			openstackCluster := OpenStackCluster().WithBastionStatus(bastionStatus).Build()
			Expect(openstackCluster.Status.Bastion).To(Equal(bastionStatus))
		})
	})
})
