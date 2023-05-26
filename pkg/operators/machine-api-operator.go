package operators

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var (
	maoDeployment        = "machine-api-operator"
	maoManagedDeployment = "machine-api-controllers"
)

var _ = Describe(
	"Machine API operator deployment should",
	framework.LabelDisruptive, framework.LabelOperators, framework.LabelMachines,
	Serial,
	func() {
		var gatherer *gatherer.StateGatherer

		BeforeEach(func() {
			var err error
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}
		})

		It("be available", func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(framework.IsDeploymentAvailable(ctx, client, maoDeployment, framework.MachineAPINamespace)).To(BeTrue())
		})

		It("reconcile controllers deployment", func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initialDeployment, err := framework.GetDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("deleting deployment %q", maoManagedDeployment))
			Expect(framework.DeleteDeployment(ctx, client, initialDeployment)).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(ctx, client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())
		})

		It("maintains deployment spec", func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initialDeployment, err := framework.GetDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			changedDeployment := initialDeployment.DeepCopy()
			changedDeployment.Spec.Replicas = pointer.Int32(0)

			By(fmt.Sprintf("updating deployment %q", maoManagedDeployment))
			Expect(framework.UpdateDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace, changedDeployment)).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(ctx, client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		})

		It("reconcile mutating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())
		})

		It("reconcile validating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())
		})

		It("recover after validating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			// Record the UID of the current ValidatingWebhookConfiguration
			initial, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())
			initialUID := initial.GetUID()

			Expect(framework.DeleteValidatingWebhookConfiguration(ctx, client, initial)).To(Succeed())

			// Ensure that either UID changes (to show a new object) or that the existing object is gone
			key := runtimeclient.ObjectKey{Name: initial.Name}
			Eventually(func() (apitypes.UID, error) {
				current := &admissionregistrationv1.ValidatingWebhookConfiguration{}
				if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
					return "", err
				}

				return current.GetUID(), nil
			}).ShouldNot(Equal(initialUID))

			// Ensure that a new object has been created and matches expectations
			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())
		})

		It("recover after mutating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			// Record the UID of the current MutatingWebhookConfiguration
			initial, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())
			initialUID := initial.GetUID()

			Expect(framework.DeleteMutatingWebhookConfiguration(ctx, client, initial)).To(Succeed())

			// Ensure that either UID changes (to show a new object) or that the existing object is gone
			key := runtimeclient.ObjectKey{Name: initial.Name}
			Eventually(func() (apitypes.UID, error) {
				current := &admissionregistrationv1.MutatingWebhookConfiguration{}
				if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
					return "", err
				}

				return current.GetUID(), nil
			}).ShouldNot(Equal(initialUID))

			// Ensure that a new object has been created and matches expectations
			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())
		})

		It("maintains spec after mutating webhook configuration change and preserve caBundle", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			initial, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateMutatingWebhookConfiguration(ctx, client, toUpdate)).To(Succeed())

			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())

			updated, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).ToNot(BeNil())

			for i, webhook := range updated.Webhooks {
				Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
			}
		})

		It("maintains spec after validating webhook configuration change and preserve caBundle", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			ctx := framework.GetContext()

			initial, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateValidatingWebhookConfiguration(ctx, client, toUpdate)).To(Succeed())

			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue())

			updated, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).ToNot(BeNil())

			for i, webhook := range updated.Webhooks {
				Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
			}
		})
	})

var _ = Describe(
	"Machine API cluster operator status should", framework.LabelOperators, framework.LabelMachines, func() {
		It("be available", func() {
			ctx := framework.GetContext()

			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			Expect(framework.WaitForStatusAvailableShort(ctx, client, "machine-api")).To(BeTrue())
		})
	})

var _ = Describe(
	"When cluster-wide proxy is configured, Machine API cluster operator should ",
	framework.LabelDisruptive, framework.LabelOperators, framework.LabelPeriodic, framework.LabelMachines,
	Serial,
	func() {
		var gatherer *gatherer.StateGatherer
		client, err := framework.LoadClient()
		ctx := framework.GetContext()
		Expect(err).NotTo(HaveOccurred())

		BeforeEach(func() {
			var err error
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())

			By("deploying an HTTP proxy")
			framework.DeployProxy(client)

			By("configuring cluster-wide proxy")
			framework.ConfigureClusterWideProxy(client)
		})

		// Machines required for test: 1
		// Reason: Tests that machine creation is possible behind a proxy.
		It("create machines when configured behind a proxy", func() {
			By("creating a machineset")
			machineSet, err := framework.CreateMachineSet(client, framework.BuildMachineSetParams(ctx, client, 1))
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the all MachineSet's Machines (and Nodes) to become Running (and Ready)")
			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			By("destroying a machineset")
			Expect(client.Delete(context.Background(), machineSet)).To(Succeed())
			framework.WaitForMachineSetsDeleted(ctx, client, machineSet)
		})

		AfterEach(func() {
			By("unconfiguring cluster-wide proxy")
			framework.UnconfigureClusterWideProxy(client)

			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}

			By("waiting for MAO, KAPI and KCM cluster operators to become available")
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			By("waiting for KAPI cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-apiserver")).To(BeTrue())

			By("waiting for KCM cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-controller-manager")).To(BeTrue())

			By("waiting for MAO cluster operator to become available")
			Expect(framework.WaitForStatusAvailableMedium(ctx, client, "machine-api")).To(BeTrue())

			By("Removing the mitm-proxy")
			framework.DeleteProxy(client)
		})
	})
