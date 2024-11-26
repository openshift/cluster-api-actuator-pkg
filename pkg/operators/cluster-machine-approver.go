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

var _ = Describe("Cluster Machine Approver deployment", framework.LabelMachineApprover, framework.LabelLEVEL0, func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.IsDeploymentAvailable(client, cmaDeployment, cmaNamespace)).To(BeTrue())
	})
})

var _ = Describe("Cluster Machine Approver Cluster Operator Status", framework.LabelMachineApprover, framework.LabelLEVEL0, func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		Expect(framework.WaitForStatusAvailableShort(client, cmaClusterOperator)).To(BeTrue())
	})
})
