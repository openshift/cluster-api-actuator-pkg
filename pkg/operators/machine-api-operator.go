package operators

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	maoDeployment        = "machine-api-operator"
	maoManagedDeployment = "machine-api-controllers"
)

// Webhooks may take up to 10 minutes to sync. When changed we first have to wait for a full Deployment
// rollout (5 minutes) and DaemonSet rollout (another 5 minutes) to occur before they are synced.
var defaultReconcileTime = 10 * time.Minute

var _ = Describe("[Feature:Operators] Machine API operator deployment should", func() {
	defer GinkgoRecover()
	var client client.Client

	BeforeEach(func() {
		var err error
		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("checking deployment %q is available", maoDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoDeployment,
			framework.MachineAPINamespace, framework.WaitLong)).To(BeTrue())
	})

	It("reconcile controllers deployment", func() {
		initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())

		By(fmt.Sprintf("deleting deployment %q", maoManagedDeployment))
		Expect(framework.DeleteDeployment(client, initialDeployment, defaultReconcileTime)).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())

		By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
		Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())
	})

	It("maintains deployment spec", func() {
		initialDeployment, err := framework.GetDeployment(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())

		changedDeployment := initialDeployment.DeepCopy()
		changedDeployment.Spec.Replicas = pointer.Int32Ptr(0)

		By(fmt.Sprintf("updating deployment %q", maoManagedDeployment))
		Expect(framework.UpdateDeployment(client, maoManagedDeployment, framework.MachineAPINamespace, changedDeployment, defaultReconcileTime)).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q spec matches", maoManagedDeployment))
		Expect(framework.IsDeploymentSynced(client, initialDeployment, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())

		By(fmt.Sprintf("checking deployment %q is available again", maoManagedDeployment))
		Expect(framework.IsDeploymentAvailable(client, maoManagedDeployment, framework.MachineAPINamespace, defaultReconcileTime)).To(BeTrue())

	})

	It("reconcile mutating webhook configuration", func() {
		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("reconcile validating webhook configuration", func() {
		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("recover after validating webhook configuration deletion", func() {
		Expect(framework.DeleteValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration)).To(Succeed())
		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("recover after mutating webhook configuration deletion", func() {
		Expect(framework.DeleteMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration)).To(Succeed())
		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())
	})

	It("maintains spec after mutating webhook configuration change and preserve caBundle", func() {
		initial, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
		Expect(initial).ToNot(BeNil())
		Expect(err).To(Succeed())

		toUpdate := initial.DeepCopy()
		for _, webhook := range toUpdate.Webhooks {
			webhook.ClientConfig.CABundle = []byte("test")
			webhook.AdmissionReviewVersions = []string{"test"}
		}

		Expect(framework.UpdateMutatingWebhookConfiguration(client, toUpdate)).To(Succeed())

		Expect(framework.IsMutatingWebhookConfigurationSynced(client)).To(BeTrue())

		updated, err := framework.GetMutatingWebhookConfiguration(client, framework.DefaultMutatingWebhookConfiguration.Name)
		Expect(updated).ToNot(BeNil())
		Expect(err).To(Succeed())

		for i, webhook := range updated.Webhooks {
			Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
		}
	})

	It("maintains spec after validating webhook configuration change and preserve caBundle", func() {
		initial, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
		Expect(initial).ToNot(BeNil())
		Expect(err).To(Succeed())

		toUpdate := initial.DeepCopy()
		for _, webhook := range toUpdate.Webhooks {
			webhook.ClientConfig.CABundle = []byte("test")
			webhook.AdmissionReviewVersions = []string{"test"}
		}

		Expect(framework.UpdateValidatingWebhookConfiguration(client, toUpdate)).To(Succeed())

		Expect(framework.IsValidatingWebhookConfigurationSynced(client)).To(BeTrue())

		updated, err := framework.GetValidatingWebhookConfiguration(client, framework.DefaultValidatingWebhookConfiguration.Name)
		Expect(updated).ToNot(BeNil())
		Expect(err).To(Succeed())

		for i, webhook := range updated.Webhooks {
			Expect(webhook.ClientConfig.CABundle).To(Equal(initial.Webhooks[i].ClientConfig.CABundle))
		}
	})

})

var _ = Describe("[Feature:Operators] Machine API cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsStatusAvailable(client, "machine-api")).To(BeTrue())
	})
})
