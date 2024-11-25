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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capibmv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("IBMPowerVSCluster", func() {
	Describe("Build", func() {
		It("should return a default IBMPowerVSCluster when no options are specified", func() {
			powerVSCluster := PowerVSCluster().Build()
			Expect(powerVSCluster).ToNot(BeNil())
			Expect(powerVSCluster.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(powerVSCluster.TypeMeta.Kind).To(Equal("IBMPowerVSCluster"))
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			powerVSCluster := PowerVSCluster().WithAnnotations(annotations).Build()
			Expect(powerVSCluster.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			powerVSCluster := PowerVSCluster().WithCreationTimestamp(timestamp).Build()
			Expect(powerVSCluster.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			powerVSCluster := PowerVSCluster().WithDeletionTimestamp(&timestamp).Build()
			Expect(powerVSCluster.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-"
			powerVSCluster := PowerVSCluster().WithGenerateName(generateName).Build()
			Expect(powerVSCluster.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			powerVSCluster := PowerVSCluster().WithLabels(labels).Build()
			Expect(powerVSCluster.Labels).To(Equal(labels))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-powervs-cluster"
			powerVSCluster := PowerVSCluster().WithName(name).Build()
			Expect(powerVSCluster.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithNamespace("ns").Build()
			Expect(powerVSCluster.Namespace).To(Equal("ns"))
		})
	})

	// Spec fields.

	Describe("WithControlPlaneEndpoint", func() {
		endpoint := clusterv1.APIEndpoint{Host: "example.com", Port: 6443}
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithControlPlaneEndpoint(endpoint).Build()
			Expect(powerVSCluster.Spec.ControlPlaneEndpoint).To(Equal(endpoint))
		})
	})

	Describe("WithLoadBalancer", func() {
		loadBalancers := []capibmv1.VPCLoadBalancerSpec{{Name: "loadBalancer"}}
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithLoadBalancer(loadBalancers).Build()
			Expect(powerVSCluster.Spec.LoadBalancers).To(Equal(loadBalancers))
		})
	})

	Describe("WithNetwork", func() {
		network := capibmv1.IBMPowerVSResourceReference{Name: ptr.To("network-name")}
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithNetwork(network).Build()
			Expect(powerVSCluster.Spec.Network).To(Equal(network))
		})
	})

	Describe("WithResourceGroup", func() {
		resourceGroup := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("resource-group")}
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithResourceGroup(resourceGroup).Build()
			Expect(powerVSCluster.Spec.ResourceGroup).To(Equal(resourceGroup))
		})
	})

	Describe("WithServiceInstance", func() {
		serviceInstance := &capibmv1.IBMPowerVSResourceReference{Name: ptr.To("service-instance")}
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithServiceInstance(serviceInstance).Build()
			Expect(powerVSCluster.Spec.ServiceInstance).To(Equal(serviceInstance))
		})
	})

	Describe("WithZone", func() {
		zone := ptr.To("test-zone")
		It("should return the custom value when specified", func() {
			powerVSCluster := PowerVSCluster().WithZone(zone).Build()
			Expect(powerVSCluster.Spec.Zone).To(Equal(zone))
		})
	})

	// Status fields.

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := clusterv1.Conditions{{Type: "Ready", Status: "True"}}
			powerVSCluster := PowerVSCluster().WithConditions(conditions).Build()
			Expect(powerVSCluster.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithReady", func() {
		It("should return the custom value when specified", func() {
			ready := true
			powerVSCluster := PowerVSCluster().WithReady(ready).Build()
			Expect(powerVSCluster.Status.Ready).To(Equal(ready))
		})
	})

})
