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
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

var _ = Describe("MachineSet", func() {
	Describe("Build", func() {
		It("should return a default machine set when no options are specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet).ToNot(BeNil())
		})
	})

	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			machineSet := MachineSet().WithAnnotations(annotations).Build()
			Expect(machineSet.Annotations).To(Equal(annotations))
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithAnnotations(nil).Build()
			Expect(machineSet.Annotations).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Annotations).To(BeNil())
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			machineSet := MachineSet().WithCreationTimestamp(timestamp).Build()
			Expect(machineSet.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			machineSet := MachineSet().WithDeletionTimestamp(&timestamp).Build()
			Expect(machineSet.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-machine-set-"
			machineSet := MachineSet().WithGenerateName(generateName).Build()
			Expect(machineSet.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key1": "value1", "key2": "value2"}
			machineSet := MachineSet().WithLabels(labels).Build()
			for key, value := range labels {
				Expect(machineSet.Labels[key]).To(Equal(value))
			}
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithLabels(nil).Build()
			Expect(machineSet.Labels).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Labels).To(BeNil())
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-machine-set"
			machineSet := MachineSet().WithName(name).Build()
			Expect(machineSet.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			machineSet := MachineSet().WithNamespace("ns-test2").Build()
			Expect(machineSet.Namespace).To(Equal("ns-test2"))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerReferences := []metav1.OwnerReference{
				{APIVersion: "v1", Kind: "Cluster", Name: "test-cluster"},
			}
			machineSet := MachineSet().WithOwnerReferences(ownerReferences).Build()
			Expect(machineSet.OwnerReferences).To(HaveLen(len(ownerReferences)))
		})
	})

	// Spec fields.

	Describe("WithClusterName", func() {
		It("should return the custom value when specified", func() {
			clusterName := "test-cluster"
			machineSet := MachineSet().WithClusterName(clusterName).Build()
			Expect(machineSet.Spec.ClusterName).To(Equal(clusterName))
		})
	})

	Describe("WithDeletionOrder", func() {
		It("should return the custom value when specified", func() {
			deletionOrder := clusterv1.RandomMachineSetDeletionOrder
			machineSet := MachineSet().WithDeletionOrder(deletionOrder).Build()
			Expect(machineSet.Spec.Deletion.Order).To(Equal(deletionOrder))
		})
	})

	Describe("WithMinReadySeconds", func() {
		It("should return the custom value when specified", func() {
			minReadySeconds := int32(10)
			machineSet := MachineSet().WithMinReadySeconds(minReadySeconds).Build()
			Expect(ptr.Deref(machineSet.Spec.Template.Spec.MinReadySeconds, 0)).To(Equal(minReadySeconds))
		})
	})

	Describe("WithReplicas", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(5)
			machineSet := MachineSet().WithReplicas(replicas).Build()
			Expect(machineSet.Spec.Replicas).To(Equal(&replicas))
		})
	})

	Describe("WithSelector", func() {
		It("should return the custom value when specified", func() {
			selector := metav1.LabelSelector{
				MatchLabels: map[string]string{"key": "value"},
			}
			machineSet := MachineSet().WithSelector(selector).Build()
			Expect(machineSet.Spec.Selector.MatchLabels["key"]).To(Equal("value"))
		})
	})

	Describe("WithTemplate", func() {
		It("should return the custom value when specified", func() {
			template := clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{"key": "value"},
				},
			}
			machineSet := MachineSet().WithTemplate(template).Build()
			Expect(machineSet.Spec.Template.Labels["key"]).To(Equal("value"))
		})
	})

	// Status fields.

	Describe("WithStatusConditions", func() {
		It("should return the custom value when specified and not nil", func() {
			conditions := []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			}
			machineSet := MachineSet().WithStatusConditions(conditions).Build()
			Expect(machineSet.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithStatusV1Beta1AvailableReplicas", func() {
		It("should return the custom value when specified", func() {
			availableReplicas := int32(5)
			machineSet := MachineSet().WithStatusV1Beta1AvailableReplicas(availableReplicas).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(machineSet.Status.Deprecated.V1Beta1.AvailableReplicas).To(Equal(availableReplicas))
		})
	})

	Describe("WithStatusV1Beta1Conditions", func() {
		It("should return the custom value when specified and not nil", func() {
			conditions := clusterv1.Conditions{
				{Type: "Ready", Status: corev1.ConditionTrue},
			}
			machineSet := MachineSet().WithStatusV1Beta1Conditions(conditions).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(machineSet.Status.Deprecated.V1Beta1.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithStatusV1Beta1FailureMessage", func() {
		It("should return the custom value when specified and not nil", func() {
			message := "test error"
			machineSet := MachineSet().WithStatusV1Beta1FailureMessage(message).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*machineSet.Status.Deprecated.V1Beta1.FailureMessage).To(Equal(message))
		})
	})

	Describe("WithStatusV1Beta1FailureReason", func() {
		It("should return the custom value when specified and not nil", func() {
			reason := capierrors.InvalidConfigurationMachineSetError
			machineSet := MachineSet().WithStatusV1Beta1FailureReason(reason).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*machineSet.Status.Deprecated.V1Beta1.FailureReason).To(Equal(reason))
		})
	})

	Describe("WithStatusV1Beta1FullyLabeledReplicas", func() {
		It("should return the custom value when specified", func() {
			fullyLabeledReplicas := int32(5)
			machineSet := MachineSet().WithStatusV1Beta1FullyLabeledReplicas(fullyLabeledReplicas).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(machineSet.Status.Deprecated.V1Beta1.FullyLabeledReplicas).To(Equal(fullyLabeledReplicas))
		})
	})

	Describe("WithStatusObservedGeneration", func() {
		It("should return the custom value when specified", func() {
			observedGeneration := int64(1)
			machineSet := MachineSet().WithStatusObservedGeneration(observedGeneration).Build()
			Expect(machineSet.Status.ObservedGeneration).To(Equal(observedGeneration))
		})
	})

	Describe("WithStatusV1Beta1ReadyReplicas", func() {
		It("should return the custom value when specified", func() {
			readyReplicas := int32(5)
			machineSet := MachineSet().WithStatusV1Beta1ReadyReplicas(readyReplicas).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(machineSet.Status.Deprecated.V1Beta1.ReadyReplicas).To(Equal(readyReplicas))
		})
	})

	Describe("WithStatusReplicas", func() {
		It("should return the custom value when specified", func() {
			repliacs := int32(5)
			machineSet := MachineSet().WithStatusReplicas(repliacs).Build()
			Expect(ptr.Deref(machineSet.Status.Replicas, 0)).To(Equal(repliacs))
		})
	})

	Describe("WithStatusSelector", func() {
		It("should return the custom value when specified", func() {
			statusSelector := "test-selector"
			machineSet := MachineSet().WithStatusSelector(statusSelector).Build()
			Expect(machineSet.Status.Selector).To(Equal(statusSelector))
		})
	})

})
