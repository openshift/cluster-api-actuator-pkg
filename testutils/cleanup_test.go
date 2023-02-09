/*
Copyright 2022 Red Hat, Inc.

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

package testutils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	corev1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/core/v1"
	machinev1beta1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/machine/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
)

var _ = Describe("Cleanup", func() {
	var namespaceName string

	BeforeEach(func() {
		By("Setting up a namespace for the test")
		ns := corev1resourcebuilder.Namespace().WithGenerateName("test-utils-").Build()
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		namespaceName = ns.GetName()

		// Creating some Machines to cleanup
		machineBuilder := machinev1beta1resourcebuilder.Machine().
			WithGenerateName("cleanup-resources-test-").
			WithNamespace(namespaceName)

		for i := 0; i < 3; i++ {
			Expect(k8sClient.Create(ctx, machineBuilder.Build())).To(Succeed())
		}
	})

	It("should delete all Machines in the namespace", func() {
		CleanupResources(Default, ctx, cfg, k8sClient, namespaceName,
			&machinev1beta1.Machine{},
		)
		Expect(komega.ObjectList(&machinev1beta1.MachineList{}, client.InNamespace(namespaceName))()).To(HaveField("Items", HaveLen(0)))
	})

	It("should delete the namespace when given", func() {
		CleanupResources(Default, ctx, cfg, k8sClient, namespaceName,
			&machinev1beta1.Machine{},
		)

		ns := corev1resourcebuilder.Namespace().WithName(namespaceName).Build()
		namespaceNotFound := apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, namespaceName)

		Expect(komega.Get(ns)()).To(MatchError(namespaceNotFound))
	})

	It("should not error when no namespace is given", func() {
		// In this case it won't actually delete anything, but that shouldn't cause any errors.
		// Any remaining resources won't affect other tests as they are in a separate namespace.
		CleanupResources(Default, ctx, cfg, k8sClient, "")
	})

	It("should ignore resources in another namespace", func() {
		ns := corev1resourcebuilder.Namespace().WithGenerateName("test-utils-").Build()
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		// Creating some Machines to cleanup
		machineBuilder := machinev1beta1resourcebuilder.Machine().
			WithGenerateName("cleanup-resources-test-").
			WithNamespace(ns.GetName())

		for i := 0; i < 3; i++ {
			Expect(k8sClient.Create(ctx, machineBuilder.Build())).To(Succeed())
		}

		// Check that it can delete all resources in the namespace
		CleanupResources(Default, ctx, cfg, k8sClient, namespaceName,
			&machinev1beta1.Machine{},
		)
		Expect(komega.ObjectList(&machinev1beta1.MachineList{}, client.InNamespace(namespaceName))()).To(HaveField("Items", HaveLen(0)))

		// Check that it didn't delete anything in the other namespace
		Expect(komega.ObjectList(&machinev1beta1.MachineList{}, client.InNamespace(ns.GetName()))()).To(HaveField("Items", HaveLen(3)))

		// Cleanup the second namespace
		CleanupResources(Default, ctx, cfg, k8sClient, ns.GetName(),
			&machinev1beta1.Machine{},
		)
		Expect(komega.ObjectList(&machinev1beta1.MachineList{}, client.InNamespace(ns.GetName()))()).To(HaveField("Items", HaveLen(0)))
	})

	It("should be able to remove objects with finalizers", func() {
		machineList := &machinev1beta1.MachineList{}
		Expect(k8sClient.List(ctx, machineList, client.InNamespace(namespaceName))).To(Succeed())

		for _, m := range machineList.Items {
			machine := m.DeepCopy()

			Eventually(komega.Update(machine, func() {
				machine.SetFinalizers([]string{"finalizer1", "finalizer2"})
			})).Should(Succeed())

			Eventually(komega.Object(machine)).Should(HaveField("ObjectMeta.Finalizers", ConsistOf("finalizer1", "finalizer2")))
		}

		CleanupResources(Default, ctx, cfg, k8sClient, namespaceName,
			&machinev1beta1.Machine{},
		)
		Eventually(komega.ObjectList(&machinev1beta1.MachineList{}, client.InNamespace(namespaceName))).Should(HaveField("Items", HaveLen(0)))
	})
})
