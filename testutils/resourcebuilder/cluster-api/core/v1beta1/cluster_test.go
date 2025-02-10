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

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

var _ = Describe("Cluster", func() {
	Describe("Build", func() {
		It("should return a default cluster when no options are specified", func() {
			cluster := Cluster().Build()
			Expect(cluster).ToNot(BeNil())
		})
	})
	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			cluster := Cluster().WithAnnotations(annotations).Build()
			Expect(cluster.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			cluster := Cluster().WithCreationTimestamp(timestamp).Build()
			Expect(cluster.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			cluster := Cluster().WithDeletionTimestamp(&timestamp).Build()
			Expect(cluster.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-cluster-"
			cluster := Cluster().WithGenerateName(generateName).Build()
			Expect(cluster.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			cluster := Cluster().WithLabels(labels).Build()
			Expect(cluster.Labels).To(Equal(labels))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-cluster"
			cluster := Cluster().WithName(name).Build()
			Expect(cluster.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithNamespace("tst-ns").Build()
			Expect(cluster.Namespace).To(Equal("tst-ns"))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerRefs := []metav1.OwnerReference{
				{
					APIVersion: "cluster.x-k8s.io/v1beta1",
					Kind:       "Cluster",
					Name:       "parent-cluster",
					UID:        "12345",
				},
			}
			cluster := Cluster().WithOwnerReferences(ownerRefs).Build()
			Expect(cluster.OwnerReferences).To(Equal(ownerRefs))
		})
	})

	// Spec fields.

	Describe("WithClusterNetwork", func() {
		It("should return the custom value when specified", func() {
			network := &capiv1.ClusterNetwork{
				APIServerPort: ptr.To(int32(6443)),
				ServiceDomain: "cluster.local",
			}
			cluster := Cluster().WithClusterNetwork(network).Build()
			Expect(cluster.Spec.ClusterNetwork).To(Equal(network))
		})
	})

	Describe("WithControlPlaneEndpoint", func() {
		It("should return the custom value when specified", func() {
			endpoint := capiv1.APIEndpoint{
				Host: "api.example.com",
				Port: 6443,
			}
			cluster := Cluster().WithControlPlaneEndpoint(endpoint).Build()
			Expect(cluster.Spec.ControlPlaneEndpoint).To(Equal(endpoint))
		})
	})

	Describe("WithControlPlaneRef", func() {
		It("should return the custom value when specified", func() {
			ref := &corev1.ObjectReference{
				Kind: "KubeadmControlPlane",
				Name: "test-control-plane",
			}
			cluster := Cluster().WithControlPlaneRef(ref).Build()
			Expect(cluster.Spec.ControlPlaneRef).To(Equal(ref))
		})
	})

	Describe("WithInfrastructureRef", func() {
		It("should return the custom value when specified", func() {
			ref := &corev1.ObjectReference{
				Kind: "AWSCluster",
				Name: "test-aws-cluster",
			}
			cluster := Cluster().WithInfrastructureRef(ref).Build()
			Expect(cluster.Spec.InfrastructureRef).To(Equal(ref))
		})
	})

	Describe("WithPaused", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithPaused(true).Build()
			Expect(cluster.Spec.Paused).To(BeTrue())
		})
	})

	Describe("WithTopology", func() {
		It("should return the custom value when specified", func() {
			topology := &capiv1.Topology{
				Class: "test-class",
			}
			cluster := Cluster().WithTopology(topology).Build()
			Expect(cluster.Spec.Topology).To(Equal(topology))
		})
	})

	// Status fields.

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := capiv1.Conditions{
				{
					Type:   capiv1.ReadyCondition,
					Status: corev1.ConditionTrue,
				},
			}
			cluster := Cluster().WithConditions(conditions).Build()
			Expect(cluster.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithControlPlaneReady", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithControlPlaneReady(true).Build()
			Expect(cluster.Status.ControlPlaneReady).To(BeTrue())
		})
	})

	Describe("WithFailureDomains", func() {
		It("should return the custom value when specified", func() {
			failureDomains := capiv1.FailureDomains{
				"us-east-1a": capiv1.FailureDomainSpec{
					ControlPlane: true,
				},
			}
			cluster := Cluster().WithFailureDomains(failureDomains).Build()
			Expect(cluster.Status.FailureDomains).To(Equal(failureDomains))
		})
	})

	Describe("WithFailureMessage", func() {
		It("should return the custom value when specified", func() {
			message := "Test failure message"
			cluster := Cluster().WithFailureMessage(message).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*cluster.Status.FailureMessage).To(Equal(message))
		})
	})

	Describe("WithFailureReason", func() {
		It("should return the custom value when specified", func() {
			reason := capierrors.InvalidConfigurationClusterError
			cluster := Cluster().WithFailureReason(reason).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*cluster.Status.FailureReason).To(Equal(reason))
		})
	})

	Describe("WithInfrastructureReady", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithInfrastructureReady(true).Build()
			Expect(cluster.Status.InfrastructureReady).To(BeTrue())
		})
	})

	Describe("WithObservedGeneration", func() {
		It("should return the custom value when specified", func() {
			generation := int64(2)
			cluster := Cluster().WithObservedGeneration(generation).Build()
			Expect(cluster.Status.ObservedGeneration).To(Equal(generation))
		})
	})

	Describe("WithPhase", func() {
		It("should return the custom value when specified", func() {
			phase := "Provisioning"
			cluster := Cluster().WithPhase(phase).Build()
			Expect(cluster.Status.Phase).To(Equal(phase))
		})
	})

})
