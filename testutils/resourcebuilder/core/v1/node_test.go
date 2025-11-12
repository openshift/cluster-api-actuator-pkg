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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Node", func() {
	Describe("Build", func() {
		It("should return a default node when no options are specified", func() {
			node := Node().Build()
			Expect(node).ToNot(BeNil())
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			node := Node().WithCreationTimestamp(timestamp).Build()
			Expect(node.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			node := Node().WithDeletionTimestamp(&timestamp).Build()
			Expect(node.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-"
			node := Node().WithGenerateName(generateName).Build()
			Expect(node.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithLabel", func() {
		It("should return the custom value when specified", func() {
			node := Node().WithLabel("key", "value").Build()
			Expect(node.Labels).To(HaveKeyWithValue("key", "value"))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key1": "value1", "key2": "value2"}
			node := Node().WithLabels(labels).Build()
			Expect(node.Labels).To(Equal(labels))
		})

		It("should return nil when specified as such", func() {
			node := Node().WithLabels(nil).Build()
			Expect(node.Labels).To(BeNil())
		})

		It("should return nil when not specified", func() {
			node := Node().Build()
			Expect(node.Labels).To(BeNil())
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-name"
			node := Node().WithName(name).Build()
			Expect(node.Name).To(Equal(name))
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			}
			node := Node().WithConditions(conditions).Build()
			Expect(node.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("AsReady", func() {
		It("should set node status to ready", func() {
			node := Node().AsReady().Build()
			Expect(node.Status.Conditions).To(HaveLen(1))
			Expect(node.Status.Conditions[0].Type).To(Equal(corev1.NodeReady))
			Expect(node.Status.Conditions[0].Status).To(Equal(corev1.ConditionTrue))
		})
	})

	Describe("AsNotReady", func() {
		It("should set node status to not ready", func() {
			node := Node().AsNotReady().Build()
			Expect(node.Status.Conditions).To(HaveLen(1))
			Expect(node.Status.Conditions[0].Type).To(Equal(corev1.NodeReady))
			Expect(node.Status.Conditions[0].Status).To(Equal(corev1.ConditionFalse))
		})
	})

	Describe("AsWorker", func() {
		It("should set worker role label", func() {
			node := Node().AsWorker().Build()
			Expect(node.Labels).To(HaveKey("node-role.kubernetes.io/worker"))
		})
	})

	Describe("AsMaster", func() {
		It("should set master role label", func() {
			node := Node().AsMaster().Build()
			Expect(node.Labels).To(HaveKey("node-role.kubernetes.io/master"))
		})
	})
})
