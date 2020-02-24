package operators

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
)

var (
	deploymentDeprecatedName = "clusterapi-manager-controllers"
)

var _ = Describe("[Feature:Operators] Machine API operator deployment should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsDeploymentAvailable(client, "machine-api-operator")).To(BeTrue())
	})

	It("reconcile controllers deployment", func() {
		var err error
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		deploymentName := "machine-api-controllers"
		initialDeployment, err := framework.GetDeployment(client, deploymentName)
		if err != nil {
			initialDeployment, err = framework.GetDeployment(client, deploymentDeprecatedName)
			Expect(err).NotTo(HaveOccurred())
			deploymentName = deploymentDeprecatedName
		}

		By(fmt.Sprintf("checking deployment %q is available", deploymentName))
		Expect(framework.IsDeploymentAvailable(client, deploymentName)).To(BeTrue())

		By(fmt.Sprintf("deleting deployment %q", deploymentName))
		err = framework.DeleteDeployment(client, initialDeployment)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available again", deploymentName))
		Expect(framework.IsDeploymentAvailable(client, deploymentName)).To(BeTrue())
	})
})

var _ = Describe("[Feature:Operators] Machine API cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(framework.IsStatusAvailable(client, "machine-api")).To(BeTrue())
	})
})
