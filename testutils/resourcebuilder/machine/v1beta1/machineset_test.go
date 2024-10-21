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
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

var _ = Describe("MachineSet", func() {
	Describe("Build", func() {
		It("should return a default machineSet when no options are specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet).ToNot(BeNil())
		})
	})

	Describe("AsWorker", func() {
		It("should set the worker role and type labels", func() {
			machineSet := MachineSet().AsWorker().Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(machineSetMachineRoleLabelName, "worker"))
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(machineSetMachineTypeLabelName, "worker"))
		})
	})

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

	Describe("WithAuthoritativeAPI", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.AuthoritativeAPI).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			authority := machinev1beta1.MachineAuthorityMachineAPI
			machineSet := MachineSet().WithAuthoritativeAPI(authority).Build()
			Expect(machineSet.Spec.AuthoritativeAPI).To(Equal(authority))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			machineSet := MachineSet().WithCreationTimestamp(timestamp).Build()
			Expect(machineSet.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletePolicy", func() {
		It("should return the custom value when specified", func() {
			policy := "Random"
			machineSet := MachineSet().WithDeletePolicy(policy).Build()
			Expect(machineSet.Spec.DeletePolicy).To(Equal(policy))
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
			generateName := "test-"
			machineSet := MachineSet().WithGenerateName(generateName).Build()
			Expect(machineSet.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabel", func() {
		It("should return the custom value when specified", func() {
			machineSet := MachineSet().WithLabel("key", "value").Build()
			Expect(machineSet.ObjectMeta.Labels).To(HaveKeyWithValue("key", "value"))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key1": "value1", "key2": "value2"}
			machineSet := MachineSet().WithLabels(labels).Build()
			Expect(machineSet.ObjectMeta.Labels).To(Equal(labels))
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithLabels(nil).Build()
			Expect(machineSet.ObjectMeta.Labels).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.ObjectMeta.Labels).To(BeNil())
		})
	})

	Describe("WithLifecycleHooks", func() {
		It("should set default when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.Spec.LifecycleHooks).To(BeZero())
		})

		It("should set the lifecycle hooks", func() {
			hooks := machinev1beta1.LifecycleHooks{
				PreDrain: []machinev1beta1.LifecycleHook{{Name: "test-hook"}},
			}
			machineSet := MachineSet().WithLifecycleHooks(hooks).Build()
			Expect(machineSet.Spec.Template.Spec.LifecycleHooks).To(Equal(hooks))
		})
	})

	Describe("WithMachineSpec", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.Spec).To(Not(BeNil()))
		})

		It("should return the custom value when specified", func() {
			spec := machinev1beta1.MachineSpec{
				ProviderID: ptr.To("test-provider-id"),
			}
			machineSet := MachineSet().WithMachineSpec(spec).Build()
			Expect(machineSet.Spec.Template.Spec).To(Equal(spec))
		})
	})

	Describe("WithMachineSpecObjectMeta", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.Spec.ObjectMeta).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			meta := machinev1beta1.ObjectMeta{
				Labels: map[string]string{"test": "label"},
			}
			machineSet := MachineSet().WithMachineSpecObjectMeta(meta).Build()
			Expect(machineSet.Spec.Template.Spec.ObjectMeta).To(Equal(meta))
		})
	})

	Describe("WithMachineSetSpecSelector", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Selector.MatchLabels).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			sel := metav1.LabelSelector{
				MatchLabels: map[string]string{"test": "label"},
			}
			machineSet := MachineSet().WithMachineSetSpecSelector(sel).Build()
			Expect(machineSet.Spec.Selector).To(Equal(sel))
		})
	})

	Describe("WithMachineTemplateAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			machineSet := MachineSet().WithMachineTemplateAnnotations(annotations).Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Annotations).To(Equal(annotations))
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithMachineTemplateAnnotations(nil).Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Annotations).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Annotations).To(BeNil())
		})
	})

	Describe("WithMachineTemplateLabel", func() {
		It("should return the custom value when specified", func() {
			machineSet := MachineSet().WithMachineTemplateLabel("key", "value").Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("key", "value"))
		})
	})

	Describe("WithMachineTemplateLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key1": "value1", "key2": "value2"}
			machineSet := MachineSet().WithMachineTemplateLabels(labels).Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(Equal(labels))
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithMachineTemplateLabels(nil).Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.ObjectMeta.Labels).To(BeNil())
		})
	})

	Describe("WithMinReadySeconds", func() {
		It("should return the custom value when specified", func() {
			minReadySeconds := int32(30)
			machineSet := MachineSet().WithMinReadySeconds(minReadySeconds).Build()
			Expect(machineSet.Spec.MinReadySeconds).To(Equal(minReadySeconds))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-machineset"
			machineSet := MachineSet().WithName(name).Build()
			Expect(machineSet.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			machineSet := MachineSet().WithNamespace("ns-test-4").Build()
			Expect(machineSet.Namespace).To(Equal("ns-test-4"))
		})
	})

	Describe("WithProviderSpec", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.Spec.ProviderSpec).To(Not(BeNil()))
		})

		It("should return the custom value when specified", func() {
			providerSpec := machinev1beta1.ProviderSpec{
				Value: &runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
			}
			machineSet := MachineSet().WithProviderSpec(providerSpec).Build()
			Expect(machineSet.Spec.Template.Spec.ProviderSpec).To(Equal(providerSpec))
		})
	})

	Describe("WithProviderSpecBuilder", func() {
		It("should return the custom value when specified", func() {
			providerSpecBuilder := AWSProviderSpec()
			machineSet := MachineSet().WithProviderSpecBuilder(providerSpecBuilder).Build()
			Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).ToNot(BeNil())
		})
	})

	Describe("WithReplicas", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(3)
			machineSet := MachineSet().WithReplicas(replicas).Build()
			Expect(*machineSet.Spec.Replicas).To(Equal(replicas))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Replicas).To(BeNil())
		})
	})

	Describe("WithTaints", func() {
		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Spec.Template.Spec.Taints).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			taints := []corev1.Taint{{Key: "test", Value: "taint", Effect: corev1.TaintEffectNoSchedule}}
			machineSet := MachineSet().WithTaints(taints).Build()
			Expect(machineSet.Spec.Template.Spec.Taints).To(Equal(taints))
		})
	})

	// Status field tests

	Describe("WithAuthoritativeAPIStatus", func() {
		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.AuthoritativeAPI).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			authority := machinev1beta1.MachineAuthorityMachineAPI
			machineSet := MachineSet().WithAuthoritativeAPIStatus(authority).Build()
			Expect(machineSet.Status.AuthoritativeAPI).To(Equal(authority))
		})
	})

	Describe("WithAvailableReplicas", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(3)
			machineSet := MachineSet().WithAvailableReplicas(replicas).Build()
			Expect(machineSet.Status.AvailableReplicas).To(Equal(replicas))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.AvailableReplicas).To(BeZero())
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := []machinev1beta1.Condition{
				{Type: "Ready", Status: corev1.ConditionTrue},
			}
			machineSet := MachineSet().WithConditions(conditions).Build()
			Expect(machineSet.Status.Conditions).To(Equal(conditions))
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.Conditions).To(BeNil())
		})

		It("should return nil when specified as such", func() {
			machineSet := MachineSet().WithConditions(nil).Build()
			Expect(machineSet.Status.Conditions).To(BeNil())
		})
	})

	Describe("WithErrorMessage", func() {
		It("should return the custom value when specified", func() {
			errorMsg := "test error"
			machineSet := MachineSet().WithErrorMessage(errorMsg).Build()
			Expect(*machineSet.Status.ErrorMessage).To(Equal(errorMsg))
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.ErrorMessage).To(BeNil())
		})
	})

	Describe("WithErrorReason", func() {
		It("should return the custom value when specified", func() {
			errorReason := machinev1beta1.InvalidConfigurationMachineSetError
			machineSet := MachineSet().WithErrorReason(errorReason).Build()
			Expect(*machineSet.Status.ErrorReason).To(Equal(errorReason))
		})

		It("should return nil when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.ErrorReason).To(BeNil())
		})
	})

	Describe("WithFullyLabeledReplicas", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(2)
			machineSet := MachineSet().WithFullyLabeledReplicas(replicas).Build()
			Expect(machineSet.Status.FullyLabeledReplicas).To(Equal(replicas))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.FullyLabeledReplicas).To(BeZero())
		})
	})

	Describe("WithObservedGeneration", func() {
		It("should return the custom value when specified", func() {
			generation := int64(5)
			machineSet := MachineSet().WithObservedGeneration(generation).Build()
			Expect(machineSet.Status.ObservedGeneration).To(Equal(generation))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.ObservedGeneration).To(BeZero())
		})
	})

	Describe("WithReadyReplicas", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(4)
			machineSet := MachineSet().WithReadyReplicas(replicas).Build()
			Expect(machineSet.Status.ReadyReplicas).To(Equal(replicas))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.ReadyReplicas).To(BeZero())
		})
	})

	Describe("WithReplicasStatus", func() {
		It("should return the custom value when specified", func() {
			replicas := int32(5)
			machineSet := MachineSet().WithReplicasStatus(replicas).Build()
			Expect(machineSet.Status.Replicas).To(Equal(replicas))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.Replicas).To(BeZero())
		})
	})

	Describe("WithSynchronizedGeneration", func() {
		It("should return the custom value when specified", func() {
			generation := int64(3)
			machineSet := MachineSet().WithSynchronizedGeneration(generation).Build()
			Expect(machineSet.Status.SynchronizedGeneration).To(Equal(generation))
		})

		It("should return the default value when not specified", func() {
			machineSet := MachineSet().Build()
			Expect(machineSet.Status.SynchronizedGeneration).To(BeZero())
		})
	})
})
