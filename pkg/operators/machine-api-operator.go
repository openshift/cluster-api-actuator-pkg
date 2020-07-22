package operators

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	maoDeployment        = "machine-api-operator"
	maoManagedDeployment = "machine-api-controllers"
)
var _ = Describe("[Feature:Operators] Machine API operator deployment should", func() {
	defer GinkgoRecover()

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
})

var _ = Describe("[Feature:Operators] Machine API cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.WaitForStatusAvailable(client, "machine-api")).To(BeTrue())
	})

	It("be degraded when a pod owned by the operator is prevented from being available", func() {
		c, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		// https://github.com/openshift/machine-api-operator/blob/d234cceb5de18b83aa0609d17db7d835f2d78973/pkg/operator/sync.go#L310-L313
		mAPIControllersLabels := map[string]string{
			"api":     "clusterapi",
			"k8s-app": "controller",
		}

		Eventually(func() (bool, error) {
			// get machine API controllers pods
			podList := &v1.PodList{}
			if err := c.List(context.TODO(), podList, client.MatchingLabels(mAPIControllersLabels)); err != nil {
				klog.Errorf("failed to list pods: %v", err)
				return false, nil
			}
			if len(podList.Items) < 1 {
				klog.Errorf("list of pods is empty")
				return false, nil
			}

			// delete machine API controllers pods
			for k := range podList.Items {
				if err := c.Delete(context.Background(), &podList.Items[k]); err != nil {
					klog.Errorf("failed to delete pod: %v", err)
					return false, nil
				}
			}

			isStatusDegraded, err := framework.IsStatusDegraded(c, "machine-api")
			if err != nil {
				return false, nil
			}

			return isStatusDegraded, nil
		}, framework.WaitLong, framework.RetryShort).Should(BeTrue())

		Expect(framework.WaitForStatusAvailable(c, "machine-api")).To(BeTrue())
	})
})
