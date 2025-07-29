package operators

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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
	framework.LabelMAPI,
	Serial,
	func() {
		var gatherer *gatherer.StateGatherer

		BeforeEach(func() {
			var err error
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
			}
		})

		It("be available", framework.LabelLEVEL0, func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")
			Expect(framework.IsDeploymentAvailable(ctx, client, maoDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed to wait for %s Deployment to become available", maoDeployment))
		})

		It("reconcile controllers deployment", framework.LabelDisruptive, func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			initialDeployment, err := framework.GetDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get %s Deployment", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed to wait for %s Deployment to become available", maoManagedDeployment))

			By(fmt.Sprintf("deleting deployment %q", maoManagedDeployment))
			Expect(framework.DeleteDeployment(ctx, client, initialDeployment)).NotTo(HaveOccurred(),
				fmt.Sprintf("Failed to delete %s Deployment", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed to wait for %s Deployment to become available", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(ctx, client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed verifying %s Deployment spec has been reconciled", maoManagedDeployment))
		})

		It("maintains deployment spec", framework.LabelDisruptive, func() {
			ctx := framework.GetContext()
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			initialDeployment, err := framework.GetDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get %s Deployment", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed to wait %s Deployment to become available", maoManagedDeployment))

			changedDeployment := initialDeployment.DeepCopy()
			changedDeployment.Spec.Replicas = ptr.To[int32](0)

			By(fmt.Sprintf("updating deployment %q", maoManagedDeployment))
			Expect(framework.UpdateDeployment(ctx, client, maoManagedDeployment, framework.MachineAPINamespace, changedDeployment)).NotTo(HaveOccurred(),
				fmt.Sprintf("Failed to update %s Deployment", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(ctx, client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed verifying %s Deployment spec has been reconciled", maoManagedDeployment))

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(ctx, client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				fmt.Sprintf("Failed to wait for %s Deployment to become available", maoManagedDeployment))

		})

		It("reconcile mutating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for MutatingWebhookConfiguration to be in sync")
		})

		It("reconcile validating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for ValidatingWebhookConfiguration to be in sync")
		})

		It("recover after validating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			// Record the UID of the current ValidatingWebhookConfiguration
			initial, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get ValidatingWebhookConfiguration")
			Expect(initial).ToNot(BeNil(), "ValidatingWebhookConfiguration should not be nil")
			initialUID := initial.GetUID()

			Expect(framework.DeleteValidatingWebhookConfiguration(ctx, client, initial)).To(Succeed(),
				"Failed to delete ValidatingWebhookConfiguration")

			// Ensure that either UID changes (to show a new object) or that the existing object is gone
			key := runtimeclient.ObjectKey{Name: initial.Name}
			Eventually(func() (apitypes.UID, error) {
				current := &admissionregistrationv1.ValidatingWebhookConfiguration{}
				if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
					return "", err
				}

				return current.GetUID(), nil
			}).ShouldNot(Equal(initialUID), "ValidatingWebhookConfiguration should have an updated UID")

			// Ensure that a new object has been created and matches expectations
			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for ValidatingWebhookConfiguration to be in sync")
		})

		It("recover after mutating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			// Record the UID of the current MutatingWebhookConfiguration
			initial, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get MutatingWebhookConfiguration")
			Expect(initial).ToNot(BeNil(), "MutatingWebhookConfiguration should not be nil")
			initialUID := initial.GetUID()

			Expect(framework.DeleteMutatingWebhookConfiguration(ctx, client, initial)).To(Succeed(), "Failed to delete MutatingWebhookConfiguration")

			// Ensure that either UID changes (to show a new object) or that the existing object is gone
			key := runtimeclient.ObjectKey{Name: initial.Name}
			Eventually(func() (apitypes.UID, error) {
				current := &admissionregistrationv1.MutatingWebhookConfiguration{}
				if err := client.Get(context.Background(), key, current); err != nil && !apierrors.IsNotFound(err) {
					return "", err
				}

				return current.GetUID(), nil
			}).ShouldNot(Equal(initialUID), "MutatingWebhookConfiguration should have an updated UID")

			// Ensure that a new object has been created and matches expectations
			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for MutatingWebhookConfiguration to be in sync")
		})

		It("maintains spec after mutating webhook configuration change and preserve caBundle", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			initial, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get MutatingWebhookConfiguration")
			Expect(initial).ToNot(BeNil(), "MutatingWebhookConfiguration should not be nil")

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateMutatingWebhookConfiguration(ctx, client, toUpdate)).To(Succeed(),
				"Failed to update MutatingWebhookConfiguration")

			Expect(framework.IsMutatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for MutatingWebhookConfiguration to be in sync")

			updated, err := framework.GetMutatingWebhookConfiguration(ctx, client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get MutatingWebhookConfiguration")
			Expect(updated).ToNot(BeNil(), "MutatingWebhookConfiguration should not be nil")

			for i, webhook := range updated.Webhooks {
				Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle),
					"MutatingWebhookConfiguration CABundles should all be equal")
			}
		})

		It("maintains spec after validating webhook configuration change and preserve caBundle", framework.LabelDisruptive, func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx := framework.GetContext()

			initial, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get ValidatingWebhookConfiguration")
			Expect(initial).ToNot(BeNil(), "ValidatingWebhookConfiguration should not be nil")

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateValidatingWebhookConfiguration(ctx, client, toUpdate)).To(Succeed(),
				"Failed to update ValidatingWebhookConfiguration")

			Expect(framework.IsValidatingWebhookConfigurationSynced(ctx, client)).To(BeTrue(),
				"Failed to wait for ValidatingWebhookConfiguration to be in sync")

			updated, err := framework.GetValidatingWebhookConfiguration(ctx, client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred(), "Failed to get ValidatingWebhookConfiguration")
			Expect(updated).ToNot(BeNil(), "ValidatingWebhookConfiguration should not be nil")

			for i, webhook := range updated.Webhooks {
				Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle),
					"Validating Webhooks CABundles should all be equal")
			}
		})
	})

var _ = Describe(
	"Machine API cluster operator status should", framework.LabelMAPI, func() {
		It("be available", framework.LabelLEVEL0, func() {
			ctx := framework.GetContext()

			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			Expect(framework.WaitForStatusAvailableShort(ctx, client, "machine-api")).To(BeTrue(),
				"Failed to wait for machine-api Cluster Operator to be available")
		})
	})

var _ = Describe(
	"When cluster-wide proxy is configured, Machine API cluster operator should ",
	framework.LabelDisruptive, framework.LabelConnectedOnly, framework.LabelPeriodic, framework.LabelMAPI,
	Serial,
	func() {
		var gatherer *gatherer.StateGatherer
		client, err := framework.LoadClient()
		ctx := framework.GetContext()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		BeforeEach(func() {
			var err error
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

			By("deploying an HTTP proxy")
			framework.DeployProxy(client)

			By("configuring cluster-wide proxy")
			framework.ConfigureClusterWideProxy(client)
		})

		// Machines required for test: 1
		// Reason: Tests that machine creation is possible behind a proxy.
		It("create machines when configured behind a proxy", framework.LabelDevOnly, func() {
			By("creating a machineset")
			machineSet, err := framework.CreateMachineSet(client, framework.BuildMachineSetParams(ctx, client, 1))
			Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet")

			By("waiting for the all MachineSet's Machines (and Nodes) to become Running (and Ready)")
			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			By("destroying a machineset")
			Expect(client.Delete(context.Background(), machineSet)).To(Succeed(), "Failed to delete MachineSet")
			framework.WaitForMachineSetsDeleted(ctx, client, machineSet)
		})

		AfterEach(func() {
			By("unconfiguring cluster-wide proxy")
			framework.UnconfigureClusterWideProxy(client)

			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
			}

			By("waiting for MAO, KAPI and KCM cluster operators to become available")
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			By("waiting for KAPI cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-apiserver")).To(BeTrue(),
				"Failed to wait for kube-apiserver Cluster Operator to become available")

			By("waiting for KCM cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-controller-manager")).To(BeTrue(),
				"Failed to wait for kube-controller-manager Cluster Operator to become available")

			By("waiting for MAO cluster operator to become available")
			Expect(framework.WaitForStatusAvailableMedium(ctx, client, "machine-api")).To(BeTrue(),
				"Failed to wait for machine-api Cluster Operator to become available")

			By("waiting for all nodes to become ready")
			Expect(framework.WaitUntilAllNodesAreReady(ctx, client)).To(Succeed(),
				"Failed to wait for all nodes to become ready")

			By("waiting for all cluster operators to become available")
			coList := &configv1.ClusterOperatorList{}
			Eventually(client.List(ctx, coList)).Should(Succeed(), "failed to list ClusterOperators.")
			for _, co := range coList.Items {
				Expect(framework.WaitForStatusAvailableOverLong(ctx, client, co.Name)).To(BeTrue(),
					"Failed to wait for %s Cluster Operator to become available", co.Name)
			}

			By("Removing the mitm-proxy")
			framework.DeleteProxy(client)
		})
	})
