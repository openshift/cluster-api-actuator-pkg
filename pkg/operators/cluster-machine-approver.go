package operators

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
)

const (
	cmaDeployment      = "machine-approver"
	cmaClusterOperator = "machine-approver"
)

var _ = Describe("[Feature:Operators] Cluster Machine Approver deployment", func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.IsDeploymentAvailable(client, cmaDeployment)).To(BeTrue())
	})
})

var _ = Describe("[Feature:Operators] Cluster Machine Approver Cluster Operator Status", func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.IsStatusAvailable(client, cmaClusterOperator)).To(BeTrue())
	})
})
