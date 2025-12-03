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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capiv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

var _ = Describe("Machine", func() {
	Describe("Build", func() {
		It("should return a default machine when no options are specified", func() {
			machine := Machine().Build()
			Expect(machine).ToNot(BeNil())
		})
	})

	Describe("Annotations", func() {
		It("should have the correct annotations when set", func() {
			annotations := map[string]string{"key": "value"}
			machine := Machine().WithAnnotations(annotations).Build()
			Expect(machine.ObjectMeta.Annotations).To(Equal(annotations))
		})
	})

	Describe("CreationTimestamp", func() {
		It("should have the correct creation timestamp when set", func() {
			timestamp := metav1.Time{Time: time.Now()}
			machine := Machine().WithCreationTimestamp(timestamp).Build()
			Expect(machine.ObjectMeta.CreationTimestamp.Time).To(Equal(timestamp.Time))
		})
	})

	Describe("DeletionTimestamp", func() {
		It("should have the correct deletion timestamp when set", func() {
			timestamp := metav1.Time{Time: time.Now()}
			machine := Machine().WithDeletionTimestamp(&timestamp).Build()
			Expect(*machine.ObjectMeta.DeletionTimestamp).To(Equal(timestamp))
		})
	})

	Describe("GenerateName", func() {
		It("should have the correct generate name when set", func() {
			generateName := "test-name"
			machine := Machine().WithGenerateName(generateName).Build()
			Expect(machine.GenerateName).To(Equal(generateName))
		})
	})

	Describe("Labels", func() {
		It("should have the correct labels when set", func() {
			labels := map[string]string{"key": "value"}
			machine := Machine().WithLabels(labels).Build()
			Expect(machine.ObjectMeta.Labels).To(Equal(labels))
		})
	})

	Describe("Name", func() {
		It("should have the correct name when set", func() {
			name := "test-name"
			machine := Machine().WithName(name).Build()
			Expect(machine.Name).To(Equal(name))
		})
	})

	Describe("Namespace", func() {
		It("should have the correct namespace when set", func() {
			machine := Machine().WithNamespace("ns-test1").Build()
			Expect(machine.Namespace).To(Equal("ns-test1"))
		})
	})

	Describe("OwnerReferences", func() {
		It("should have the correct owner references when set", func() {
			ownerRef := metav1.OwnerReference{Controller: ptr.To(true)}
			machine := Machine().WithOwnerReferences([]metav1.OwnerReference{ownerRef}).Build()
			Expect(machine.ObjectMeta.OwnerReferences).To(ContainElement(ownerRef))
		})
	})

	// Spec fields.

	Describe("Bootstrap", func() {
		It("should have the correct bootstrap settings when set", func() {
			bootstrap := capiv1.Bootstrap{}
			machine := Machine().WithBootstrap(bootstrap).Build()
			Expect(machine.Spec.Bootstrap).To(Equal(bootstrap))
		})
	})

	Describe("ClusterName", func() {
		It("should have the correct cluster name when set", func() {
			machine := Machine().WithClusterName("test-cluster").Build()
			Expect(machine.Spec.ClusterName).To(Equal("test-cluster"))
		})
	})

	Describe("FailureDomain", func() {
		It("should have the correct failure domain when set", func() {
			failureDomain := "test-dm"
			machine := Machine().WithFailureDomain(&failureDomain).Build()
			Expect(*machine.Spec.FailureDomain).To(Equal(failureDomain))
		})
	})

	Describe("InfrastructureRef", func() {
		It("should have the correct infrastructure reference when set", func() {
			infraRef := corev1.ObjectReference{Name: "test-obj-ref"}
			machine := Machine().WithInfrastructureRef(infraRef).Build()
			Expect(machine.Spec.InfrastructureRef).To(Equal(infraRef))
		})
	})

	Describe("NodeDeletionTimeout", func() {
		It("should have the correct node deletion timeout when set", func() {
			timeout := metav1.Duration{Duration: 30 * time.Second}
			machine := Machine().WithNodeDeletionTimeout(&timeout).Build()
			Expect(*machine.Spec.NodeDeletionTimeout).To(Equal(timeout))
		})
	})

	Describe("NodeDrainTimeout", func() {
		It("should have the correct node drain timeout when set", func() {
			timeout := metav1.Duration{Duration: 30 * time.Second}
			machine := Machine().WithNodeDrainTimeout(&timeout).Build()
			Expect(*machine.Spec.NodeDrainTimeout).To(Equal(timeout))
		})
	})

	Describe("NodeVolumeDetachTimeout", func() {
		It("should have the correct node volume detach timeout when set", func() {
			timeout := metav1.Duration{Duration: 30 * time.Second}
			machine := Machine().WithNodeVolumeDetachTimeout(&timeout).Build()
			Expect(*machine.Spec.NodeVolumeDetachTimeout).To(Equal(timeout))
		})
	})

	Describe("ProviderID", func() {
		It("should have the correct provider ID when set", func() {
			providerID := "test-obj-id"
			machine := Machine().WithProviderID(&providerID).Build()
			Expect(*machine.Spec.ProviderID).To(Equal(providerID))
		})
	})

	Describe("Version", func() {
		It("should have the correct version when set", func() {
			version := "1.23.0"
			machine := Machine().WithVersion(&version).Build()
			Expect(*machine.Spec.Version).To(Equal(version))
		})
	})

	// Status field tests

	Describe("NodeRef", func() {
		It("should have the correct node reference when set", func() {
			nodeRef := corev1.ObjectReference{Name: "test-node"}
			machine := Machine().WithNodeRef(&nodeRef).Build()
			Expect(*machine.Status.NodeRef).To(Equal(nodeRef))
		})
	})

	Describe("NodeInfo", func() {
		It("should have the correct node info when set", func() {
			nodeInfo := &corev1.NodeSystemInfo{}
			machine := Machine().WithNodeInfo(nodeInfo).Build()
			Expect(machine.Status.NodeInfo).To(Equal(nodeInfo))
		})
	})

	Describe("LastUpdated", func() {
		It("should have the correct last updated timestamp when set", func() {
			timestamp := metav1.Now()
			machine := Machine().WithLastUpdated(&timestamp).Build()
			Expect(*machine.Status.LastUpdated).To(Equal(timestamp))
		})
	})

	Describe("Phase", func() {
		It("should have the correct phase when set", func() {
			phase := capiv1.MachinePhaseRunning
			machine := Machine().WithPhase(phase).Build()
			Expect(machine.Status.Phase).To(Equal(string(phase)))
		})
	})

	Describe("BootstrapReady", func() {
		It("should have the correct bootstrap ready status when set", func() {
			machine := Machine().WithBootstrapReady(true).Build()
			Expect(machine.Status.BootstrapReady).To(BeTrue())
		})
	})

	Describe("InfrastructureReady", func() {
		It("should have the correct infrastructure ready status when set", func() {
			machine := Machine().WithInfrastructureReady(true).Build()
			Expect(machine.Status.InfrastructureReady).To(BeTrue())
		})
	})

	Describe("ObservedGeneration", func() {
		It("should have the correct observed generation when set", func() {
			generation := int64(1)
			machine := Machine().WithObservedGeneration(generation).Build()
			Expect(machine.Status.ObservedGeneration).To(Equal(generation))
		})
	})

	Describe("Conditions", func() {
		It("should have the correct conditions when set", func() {
			condition := capiv1.Condition{Type: "Ready", Status: corev1.ConditionTrue}
			machine := Machine().WithConditions([]capiv1.Condition{condition}).Build()
			Expect(machine.Status.Conditions).To(HaveLen(1))
		})
	})

	Describe("FailureReason", func() {
		It("should have the correct failure reason when set", func() {
			reason := capierrors.InvalidConfigurationMachineError
			machine := Machine().WithFailureReason(&reason).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*machine.Status.FailureReason).To(Equal(reason))
		})

	})

	Describe("FailureMessage", func() {
		It("should have the correct failure message when set", func() {
			message := "test-fail-msg"
			machine := Machine().WithFailureMessage(&message).Build()
			//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
			Expect(*machine.Status.FailureMessage).To(Equal(message))
		})
	})

	Describe("CertificatesExpiryDate", func() {
		It("should have the correct expiry date when set", func() {
			expiryDate := metav1.Time{Time: time.Now()}
			machine := Machine().WithCertificatesExpiryDate(&expiryDate).Build()
			Expect(*machine.Status.CertificatesExpiryDate).To(Equal(expiryDate))
		})
	})

})
