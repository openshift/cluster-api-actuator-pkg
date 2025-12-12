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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

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
			network := clusterv1.ClusterNetwork{
				APIServerPort: 6443,
				ServiceDomain: "cluster.local",
			}
			cluster := Cluster().WithClusterNetwork(network).Build()
			Expect(cluster.Spec.ClusterNetwork).To(Equal(network))
		})
	})

	Describe("WithControlPlaneEndpoint", func() {
		It("should return the custom value when specified", func() {
			endpoint := clusterv1.APIEndpoint{
				Host: "api.example.com",
				Port: 6443,
			}
			cluster := Cluster().WithControlPlaneEndpoint(endpoint).Build()
			Expect(cluster.Spec.ControlPlaneEndpoint).To(Equal(endpoint))
		})
	})

	Describe("WithControlPlaneRef", func() {
		It("should return the custom value when specified", func() {
			ref := clusterv1.ContractVersionedObjectReference{
				APIGroup: "controlplane.cluster.x-k8s.io",
				Kind:     "KubeadmControlPlane",
				Name:     "test-control-plane",
			}
			cluster := Cluster().WithControlPlaneRef(ref).Build()
			Expect(cluster.Spec.ControlPlaneRef).To(Equal(ref))
		})
	})

	Describe("WithInfrastructureRef", func() {
		It("should return the custom value when specified", func() {
			ref := clusterv1.ContractVersionedObjectReference{
				APIGroup: "infrastructure.cluster.x-k8s.io",
				Kind:     "AWSCluster",
				Name:     "test-aws-cluster",
			}
			cluster := Cluster().WithInfrastructureRef(ref).Build()
			Expect(cluster.Spec.InfrastructureRef).To(Equal(ref))
		})
	})

	Describe("WithPaused", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithPaused(true).Build()
			Expect(ptr.Deref(cluster.Spec.Paused, false)).To(BeTrue())
		})
	})

	Describe("WithTopology", func() {
		It("should return the custom value when specified", func() {
			topology := clusterv1.Topology{
				ClassRef: clusterv1.ClusterClassRef{
					Name: "test-class",
				},
			}
			cluster := Cluster().WithTopology(topology).Build()
			Expect(cluster.Spec.Topology).To(Equal(topology))
		})
	})

	// Status fields.

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := []metav1.Condition{
				{
					Type:   clusterv1.ReadyCondition,
					Status: metav1.ConditionTrue,
				},
			}
			cluster := Cluster().WithConditions(conditions).Build()
			Expect(cluster.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithV1Beta1Conditions", func() {
		It("should return the custom value when specified", func() {
			conditions := clusterv1.Conditions{
				{
					Type:   clusterv1.ReadyCondition,
					Status: corev1.ConditionTrue,
				},
			}
			cluster := Cluster().WithV1Beta1Conditions(conditions).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(cluster.Status.Deprecated.V1Beta1.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithControlPlaneInitialized", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithControlPlaneInitialized(true).Build()
			Expect(ptr.Deref(cluster.Status.Initialization.ControlPlaneInitialized, false)).To(BeTrue())
		})
	})

	Describe("WithFailureDomains", func() {
		It("should return the custom value when specified", func() {
			failureDomains := []clusterv1.FailureDomain{{
				Name:         "us-east-1a",
				ControlPlane: ptr.To(true),
			},
			}
			cluster := Cluster().WithFailureDomains(failureDomains).Build()
			Expect(cluster.Status.FailureDomains).To(Equal(failureDomains))
		})
	})

	Describe("WithV1Beta1FailureMessage", func() {
		It("should return the custom value when specified", func() {
			message := "Test failure message"
			cluster := Cluster().WithV1Beta1FailureMessage(message).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*cluster.Status.Deprecated.V1Beta1.FailureMessage).To(Equal(message))
		})
	})

	Describe("WithV1Beta1FailureReason", func() {
		It("should return the custom value when specified", func() {
			reason := capierrors.InvalidConfigurationClusterError
			cluster := Cluster().WithV1Beta1FailureReason(reason).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*cluster.Status.Deprecated.V1Beta1.FailureReason).To(Equal(reason))
		})
	})

	Describe("WithInfrastructureProvisioned", func() {
		It("should return the custom value when specified", func() {
			cluster := Cluster().WithInfrastructureProvisioned(true).Build()
			Expect(ptr.Deref(cluster.Status.Initialization.InfrastructureProvisioned, false)).To(BeTrue())
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

	Describe("WithAvailabilityGates", func() {
		It("should return the custom value when specified", func() {
			gates := []clusterv1.ClusterAvailabilityGate{
				{
					ConditionType: "CustomCondition",
				},
			}
			cluster := Cluster().WithAvailabilityGates(gates).Build()
			Expect(cluster.Spec.AvailabilityGates).To(Equal(gates))
		})
	})

	Describe("WithControlPlaneStatus", func() {
		It("should return the custom value when specified", func() {
			controlPlaneStatus := &clusterv1.ClusterControlPlaneStatus{
				DesiredReplicas: ptr.To(int32(3)),
				Replicas:        ptr.To(int32(3)),
				ReadyReplicas:   ptr.To(int32(3)),
			}
			cluster := Cluster().WithControlPlaneStatus(controlPlaneStatus).Build()
			Expect(cluster.Status.ControlPlane).To(Equal(controlPlaneStatus))
		})
	})

	Describe("WithWorkersStatus", func() {
		It("should return the custom value when specified", func() {
			workersStatus := &clusterv1.WorkersStatus{
				DesiredReplicas: ptr.To(int32(5)),
				Replicas:        ptr.To(int32(5)),
				ReadyReplicas:   ptr.To(int32(5)),
			}
			cluster := Cluster().WithWorkersStatus(workersStatus).Build()
			Expect(cluster.Status.Workers).To(Equal(workersStatus))
		})
	})

})
