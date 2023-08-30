package operators

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
)

const (
	cmaDeployment      = "machine-approver"
	cmaClusterOperator = "machine-approver"
	cmaNamespace       = "openshift-cluster-machine-approver"
)

var _ = Describe("Cluster Machine Approver deployment", framework.LabelOperators, func() {
	It("should be available", func() {
		ctx := framework.GetContext()

		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		Expect(framework.IsDeploymentAvailable(ctx, client, cmaDeployment, cmaNamespace)).To(BeTrue(),
			"Failed to wait for cluster-machine-approver Deployment to become available")
	})
})

var _ = Describe("Cluster Machine Approver Cluster Operator Status", framework.LabelOperators, func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()

		Expect(framework.WaitForStatusAvailableShort(ctx, client, cmaClusterOperator)).To(BeTrue(),
			"Failed to wait for cluster-machine-approver Cluster Operator to be available")
	})
})
