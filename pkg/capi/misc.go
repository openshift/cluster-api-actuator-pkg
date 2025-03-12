package capi

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx context.Context
)

var _ = Describe("Cluster API status values", framework.LabelCAPI, Ordered, func() {
	BeforeAll(func() {
		var err error
		cl, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Failed to get Kubernetes client")
		ctx = context.TODO()
		oc, _ := framework.NewCLI()
		framework.SkipIfNotTechPreviewNoUpgrade(oc, cl)
	})

	It("should have CoreClusterControllerAvailable condition", func() {
		// Fetch the ClusterOperator resource
		co := &configv1.ClusterOperator{}
		err := cl.Get(ctx, client.ObjectKey{Name: "cluster-api"}, co)
		Expect(err).ToNot(HaveOccurred(), "Failed to fetch cluster-api ClusterOperator")

		// Check if the status contains CoreClusterControllerAvailable
		found := false
		for _, cond := range co.Status.Conditions {
			if cond.Type == "CoreClusterControllerAvailable" && cond.Status == configv1.ConditionTrue {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "Expected CoreClusterControllerAvailable condition to be present and true")
	})
})
