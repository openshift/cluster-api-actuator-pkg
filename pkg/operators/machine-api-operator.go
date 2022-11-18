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

	v1 "github.com/openshift/api/config/v1"

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
			if specReport.Failed() == true {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}
		})

		It("be available", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(framework.IsDeploymentAvailable(client, maoDeployment, framework.MachineAPINamespace)).To(BeTrue())
		})

		It("reconcile controllers deployment", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("deleting deployment %q", maoManagedDeployment))
			Expect(framework.DeleteDeployment(client, initialDeployment)).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())
		})

		It("maintains deployment spec", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			changedDeployment := initialDeployment.DeepCopy()
			changedDeployment.Spec.Replicas = pointer.Int32Ptr(0)

			By(fmt.Sprintf("updating deployment %q", maoManagedDeployment))
			Expect(framework.UpdateDeployment(client, maoManagedDeployment, framework.MachineAPINamespace, changedDeployment)).NotTo(HaveOccurred())

			By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
			Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

			By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
			Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace)).To(BeTrue())

		})

		It("reconcile mutating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
		})

		It("reconcile validating webhook configuration", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
		})

		It("recover after validating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			// Record the UID of the current ValidatingWebhookConfiguration
			initial, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())
			initialUID := initial.GetUID()

			Expect(framework.DeleteValidatingWebhookConfiguration(client, initial)).To(Succeed())

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
			Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
		})

		It("recover after mutating webhook configuration deletion", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			// Record the UID of the current MutatingWebhookConfiguration
			initial, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())
			initialUID := initial.GetUID()

			Expect(framework.DeleteMutatingWebhookConfiguration(client, initial)).To(Succeed())

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
			Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
		})

		It("maintains spec after mutating webhook configuration change and preserve caBundle", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initial, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateMutatingWebhookConfiguration(client, toUpdate)).To(Succeed())

			Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())

			updated, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).ToNot(BeNil())

			for i, webhook := range updated.Webhooks {
				Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
			}
		})

		It("maintains spec after validating webhook configuration change and preserve caBundle", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			initial, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(initial).ToNot(BeNil())

			toUpdate := initial.DeepCopy()
			for _, webhook := range toUpdate.Webhooks {
				webhook.ClientConfig.CABundle = []byte("test")
				webhook.AdmissionReviewVersions = []string{"test"}
			}

			Expect(framework.UpdateValidatingWebhookConfiguration(client, toUpdate)).To(Succeed())

			Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())

			updated, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
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
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(framework.WaitForStatusAvailableShort(client, "machine-api")).To(BeTrue())
		})
	})

var _ = Describe(
	"When cluster-wide proxy is configured, Machine API cluster operator should ",
	framework.LabelDisruptive, framework.LabelOperators, framework.LabelPeriodic, framework.LabelMachines,
	Serial,
	func() {
		var gatherer *gatherer.StateGatherer

		BeforeEach(func() {
			var err error
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())
		})

		It("create machines when configured behind a proxy", func() {
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())

			By("deploying an HTTP proxy")
			err = framework.DeployClusterProxy(client)
			Expect(err).NotTo(HaveOccurred())

			By("configuring cluster-wide proxy")
			services, err := framework.GetServices(client, map[string]string{"app": "mitm-proxy"})
			proxy, err := framework.GetClusterProxy(client)
			Expect(err).NotTo(HaveOccurred())
			proxy.Spec.HTTPProxy = "http://" + services.Items[0].Spec.ClusterIP + ":8080"
			proxy.Spec.HTTPSProxy = "http://" + services.Items[0].Spec.ClusterIP + ":8080"
			proxy.Spec.NoProxy = ".org,.com,quay.io"
			proxy.Spec.TrustedCA = v1.ConfigMapNameReference{
				Name: "mitm-custom-pki",
			}
			err = client.Update(context.Background(), proxy)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for machine-api-controller deployment to reflect configured cluster-wide proxy")
			result, err := framework.WaitForProxyInjectionSync(client, maoManagedDeployment, framework.MachineAPINamespace, true)
			Expect(result).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())

			By("creating a machineset")
			machineSetParams := framework.BuildMachineSetParams(client, 1)
			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			framework.WaitForMachineSet(client, machineSet.GetName())

			By("destroying a machineset")
			Expect(client.Delete(context.Background(), machineSet)).To(Succeed())
			framework.WaitForMachineSetDelete(client, machineSet)

			By("unconfiguring cluster-wide proxy")
			err = client.Patch(context.Background(), proxy, runtimeclient.RawPatch(apitypes.JSONPatchType, []byte(`[
			{"op": "remove", "path": "/spec/httpProxy"},
			{"op": "remove", "path": "/spec/httpsProxy"},
			{"op": "remove", "path": "/spec/noProxy"},
			{"op": "remove", "path": "/spec/trustedCA"}
		]`)))
			Expect(err).NotTo(HaveOccurred())

			By("waiting for machine-api-controller deployment to reflect unconfigured cluster-wide proxy")
			Expect(framework.WaitForProxyInjectionSync(client, maoManagedDeployment, framework.MachineAPINamespace, false)).To(BeTrue())
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() == true {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}

			By("waiting for MAO, KAPI and KCM cluster operators to become available")
			client, err := framework.LoadClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(framework.WaitForProxyInjectionSync(client, maoManagedDeployment, framework.MachineAPINamespace, false)).To(BeTrue())

			By("waiting for KAPI cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(client, "kube-apiserver")).To(BeTrue())

			By("waiting for KCM cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(client, "kube-controller-manager")).To(BeTrue())
			Expect(framework.WaitForStatusAvailableMedium(client, "machine-api")).To(BeTrue())

			By("Removing the mitm-proxy")
			Expect(framework.DestroyClusterProxy(client)).ToNot(HaveOccurred())
		})
	})
