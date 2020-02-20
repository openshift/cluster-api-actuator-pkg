package operators

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/framework"
)

var (
	deploymentDeprecatedName = "clusterapi-manager-controllers"
)

var _ = Describe("[Feature:Operators] Machine API operator deployment should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := e2e.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(e2e.IsDeploymentAvailable(client, "machine-api-operator")).To(BeTrue())
	})

	It("reconcile controllers deployment", func() {
		var err error
		client, err := e2e.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		deploymentName := "machine-api-controllers"
		initialDeployment, err := e2e.GetDeployment(client, deploymentName)
		if err != nil {
			initialDeployment, err = e2e.GetDeployment(client, deploymentDeprecatedName)
			Expect(err).NotTo(HaveOccurred())
			deploymentName = deploymentDeprecatedName
		}

		By(fmt.Sprintf("checking deployment %q is available", deploymentName))
		Expect(e2e.IsDeploymentAvailable(client, deploymentName)).To(BeTrue())

		By(fmt.Sprintf("deleting deployment %q", deploymentName))
		err = e2e.DeleteDeployment(client, initialDeployment)
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("checking deployment %q is available again", deploymentName))
		Expect(e2e.IsDeploymentAvailable(client, deploymentName)).To(BeTrue())
	})
})

var _ = Describe("[Feature:Operators] Machine API cluster operator status should", func() {
	defer GinkgoRecover()

	It("be available", func() {
		var err error
		client, err := e2e.LoadClient()
		Expect(err).NotTo(HaveOccurred())
		Expect(isStatusAvailable(client, "machine-api")).To(BeTrue())
	})
})
