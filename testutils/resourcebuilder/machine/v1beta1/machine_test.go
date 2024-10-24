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
	"github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Machine", func() {
	Describe("Build", func() {
		It("should return a default machine when no options are specified", func() {
			machine := Machine().Build()
			Expect(machine).ToNot(BeNil())
		})
	})

	Describe("AsWorker", func() {
		It("should return the custom value when specified", func() {
			machine := Machine().AsWorker().Build()
			Expect(machine.Labels).To(HaveKeyWithValue(resourcebuilder.MachineRoleLabelName, "worker"))
			Expect(machine.Labels).To(HaveKeyWithValue(resourcebuilder.MachineTypeLabelName, "worker"))
		})
	})

	Describe("AsMaster", func() {
		It("should return the custom value when specified", func() {
			machine := Machine().AsMaster().Build()
			Expect(machine.Labels).To(HaveKeyWithValue(resourcebuilder.MachineRoleLabelName, "master"))
			Expect(machine.Labels).To(HaveKeyWithValue(resourcebuilder.MachineTypeLabelName, "master"))
		})
	})

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			machine := Machine().WithAnnotations(annotations).Build()
			Expect(machine.Annotations).To(Equal(annotations))
		})

		It("should return nil when specified as such", func() {
			machine := Machine().WithAnnotations(nil).Build()
			Expect(machine.Annotations).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Annotations).To(BeNil())
		})
	})

	Describe("WithAuthoritativeAPI", func() {
		It("should return the default value when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.AuthoritativeAPI).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			machine := Machine().WithAuthoritativeAPI(machinev1beta1.MachineAuthorityMachineAPI).Build()
			Expect(machine.Spec.AuthoritativeAPI).To(Equal(machinev1beta1.MachineAuthorityMachineAPI))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			machine := Machine().WithCreationTimestamp(timestamp).Build()
			Expect(machine.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			machine := Machine().WithDeletionTimestamp(&timestamp).Build()
			Expect(machine.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-"
			machine := Machine().WithGenerateName(generateName).Build()
			Expect(machine.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabel", func() {
		It("should return the custom value when specified", func() {
			machine := Machine().WithLabel("key", "value").Build()
			Expect(machine.Labels).To(HaveKeyWithValue("key", "value"))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key1": "value1", "key2": "value2"}
			machine := Machine().WithLabels(labels).Build()
			Expect(machine.Labels).To(Equal(labels))
		})

		It("should return nil when specified as such", func() {
			machine := Machine().WithLabels(nil).Build()
			Expect(machine.Annotations).To(BeNil())
		})

		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Annotations).To(BeNil())
		})
	})

	Describe("WithLifecycleHooks", func() {
		It("should set default LifecycleHooks when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.LifecycleHooks).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			hooks := machinev1beta1.LifecycleHooks{PreDrain: []machinev1beta1.LifecycleHook{{Name: "test-hook"}}}
			machine := Machine().WithLifecycleHooks(hooks).Build()
			Expect(machine.Spec.LifecycleHooks).To(Equal(hooks))
		})
	})

	Describe("WithMachineSpec", func() {
		It("should return the default value when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec).To(Not(BeNil()))
		})

		It("should return the custom value when specified", func() {
			machineSpec := machinev1beta1.MachineSpec{}
			machine := Machine().WithMachineSpec(machineSpec).Build()
			Expect(machine.Spec).To(Equal(machineSpec))
		})
	})

	Describe("WithMachineSpecObjectMeta", func() {
		It("should return the default value when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.ObjectMeta).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			objectMeta := machinev1beta1.ObjectMeta{Labels: map[string]string{"key": "value"}}
			machine := Machine().WithMachineSpecObjectMeta(objectMeta).Build()
			Expect(machine.Spec.ObjectMeta).To(Equal(objectMeta))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-machine"
			machine := Machine().WithName(name).Build()
			Expect(machine.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			machine := Machine().WithNamespace("ns-test-3").Build()
			Expect(machine.Namespace).To(Equal("ns-test-3"))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerRefs := []metav1.OwnerReference{
				{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
					Name:       "parent-ms",
					UID:        "12345",
				},
			}
			machine := Machine().WithOwnerReferences(ownerRefs).Build()
			Expect(machine.OwnerReferences).To(Equal(ownerRefs))
		})
	})

	Describe("WithProviderID", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.ProviderID).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			providerID := "test-provider-id"
			machine := Machine().WithProviderID(&providerID).Build()
			Expect(*machine.Spec.ProviderID).To(Equal(providerID))
		})
	})

	Describe("WithProviderSpec", func() {
		It("should return the default value when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.ProviderSpec).To(Not(BeNil()))
		})

		It("should return the custom value when specified", func() {
			providerSpec := machinev1beta1.ProviderSpec{Value: nil}
			machine := Machine().WithProviderSpec(providerSpec).Build()
			Expect(machine.Spec.ProviderSpec).To(Equal(providerSpec))
		})
	})

	Describe("WithProviderSpecBuilder", func() {
		It("should return the custom value when specified", func() {
			provideSpecBuilder := AWSProviderSpec()
			machine := Machine().WithProviderSpecBuilder(provideSpecBuilder).Build()
			Expect(machine.Spec.ProviderSpec.Value).ToNot(BeNil())
		})
	})

	Describe("WithTaints", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Spec.Taints).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			taints := []corev1.Taint{
				{Key: "a", Effect: corev1.TaintEffectNoExecute},
			}
			machine := Machine().WithTaints(taints).Build()
			Expect(machine.Spec.Taints).To(Equal(taints))
		})
	})

	// Status field tests

	Describe("WithAddresses", func() {
		It("should return the custom value when specified", func() {
			addresses := []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "192.168.0.1"}}
			machine := Machine().WithAddresses(addresses).Build()
			Expect(machine.Status.Addresses).To(Equal(addresses))
		})
	})

	Describe("WithAuthoritativeAPIStatus", func() {
		It("should return the default value when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.AuthoritativeAPI).To(BeZero())
		})

		It("should return the custom value when specified", func() {
			machine := Machine().WithAuthoritativeAPIStatus(machinev1beta1.MachineAuthorityMachineAPI).Build()
			Expect(machine.Status.AuthoritativeAPI).To(Equal(machinev1beta1.MachineAuthorityMachineAPI))
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := []machinev1beta1.Condition{{Type: "Ready", Status: corev1.ConditionTrue}}
			machine := Machine().WithConditions(conditions).Build()
			Expect(machine.Status.Conditions).To(Equal(conditions))
		})

		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.Conditions).To(BeNil())
		})

		It("should return nil when specified as such", func() {
			machine := Machine().WithConditions(nil).Build()
			Expect(machine.Status.Conditions).To(BeNil())
		})
	})

	Describe("WithErrorMessage", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.ErrorMessage).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			errorMessage := "test error"
			machine := Machine().WithErrorMessage(errorMessage).Build()
			Expect(*machine.Status.ErrorMessage).To(Equal(errorMessage))
		})
	})

	Describe("WithErrorReason", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.ErrorReason).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			errorReason := machinev1beta1.CreateMachineError
			machine := Machine().WithErrorReason(errorReason).Build()
			Expect(*machine.Status.ErrorReason).To(Equal(errorReason))
		})
	})

	Describe("WithLastOperation", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.LastOperation).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			t := "test operation"
			lastOperation := machinev1beta1.LastOperation{Description: &t}
			machine := Machine().WithLastOperation(lastOperation).Build()
			Expect(*machine.Status.LastOperation).To(Equal(lastOperation))
		})
	})

	Describe("WithLastUpdated", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.LastUpdated).To(BeNil())
		})
		It("should return the custom value when specified", func() {
			lastUpdated := metav1.Now()
			machine := Machine().WithLastUpdated(lastUpdated).Build()
			Expect(*machine.Status.LastUpdated).To(Equal(lastUpdated))
		})
	})

	Describe("WithPhase", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.Phase).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			phase := machinev1beta1.PhaseRunning
			machine := Machine().WithPhase(phase).Build()
			Expect(*machine.Status.Phase).To(Equal(phase))
		})
	})

	Describe("WithProviderStatus", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.ProviderStatus).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			providerStatus := runtime.RawExtension{Raw: []byte(`{"key":"value"}`)}
			machine := Machine().WithProviderStatus(providerStatus).Build()
			Expect(*machine.Status.ProviderStatus).To(Equal(providerStatus))
		})
	})

	Describe("WithNodeRef", func() {
		It("should return nil when not specified", func() {
			machine := Machine().Build()
			Expect(machine.Status.NodeRef).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			nodeRef := corev1.ObjectReference{Name: "test-node", Namespace: "default"}
			machine := Machine().WithNodeRef(nodeRef).Build()
			Expect(*machine.Status.NodeRef).To(Equal(nodeRef))
		})
	})
})
