package operators

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe(
	"[sig-cluster-lifecycle] Cluster API When cluster-wide proxy is configured, Cluster API provider deployments should",
	framework.LabelDisruptive, framework.LabelConnectedOnly, framework.LabelPeriodic, framework.LabelCAPI,
	Serial,
	func() {
		var (
			gatherer   *gatherer.StateGatherer
			client     runtimeclient.Client
			ctx        context.Context
			isProxyJob bool
		)

		BeforeEach(func() {
			var err error

			client, err = framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			ctx = framework.GetContext()

			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

			isProxyJob, err = framework.IsClusterProxyEnabled(ctx, client)
			Expect(err).ToNot(HaveOccurred(), "Failed to check cluster proxy configuration")

			if isProxyJob {
				By("cluster-wide proxy is already configured on proxy cluster, skipping MITM proxy deployment")
			} else {
				By("deploying an HTTP proxy")
				framework.DeployProxy(client)

				By("configuring cluster-wide proxy")
				framework.ConfigureClusterWideProxy(client)
			}
		})

		It("have proxy environment variables injected", func() {
			By("waiting for Cluster API provider deployments to reflect proxy configuration")

			deployments := &appsv1.DeploymentList{}
			Eventually(client.List(ctx, deployments,
				runtimeclient.InNamespace(framework.ClusterAPINamespace),
				runtimeclient.HasLabels{"cluster.x-k8s.io/provider"},
			)).Should(Succeed(), "timed out listing Cluster API provider Deployments.")
			Expect(deployments.Items).NotTo(BeEmpty(), "no Cluster API provider Deployments found")

			for _, deploy := range deployments.Items {
				By(fmt.Sprintf("verifying proxy env vars on Deployment %s", deploy.Name))
				framework.WaitForAllContainersProxyEnvVars(client, framework.ClusterAPINamespace, deploy.Name)
			}
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
			}

			if isProxyJob {
				By("cluster-wide proxy was pre-existing, skipping MITM proxy cleanup")
				return
			}

			By("unconfiguring cluster-wide proxy")
			framework.UnconfigureClusterWideProxy(client)

			var err error

			client, err = framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to refresh client after proxy teardown")

			By("verifying proxy env vars are removed from Cluster API provider Deployments")

			deployments := &appsv1.DeploymentList{}
			Eventually(client.List(ctx, deployments,
				runtimeclient.InNamespace(framework.ClusterAPINamespace),
				runtimeclient.HasLabels{"cluster.x-k8s.io/provider"},
			)).Should(Succeed(), "timed out listing Cluster API provider Deployments.")
			Expect(deployments.Items).NotTo(BeEmpty(), "no Cluster API provider Deployments found")

			for _, deploy := range deployments.Items {
				By(fmt.Sprintf("verifying proxy env vars removed from Deployment %s", deploy.Name))
				framework.WaitForAllContainersNoProxyEnvVars(client, framework.ClusterAPINamespace, deploy.Name)
			}

			By("waiting for KAPI cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-apiserver")).To(BeTrue(),
				"Failed to wait for kube-apiserver Cluster Operator to become available")

			By("waiting for KCM cluster operator to become available")
			Expect(framework.WaitForStatusAvailableOverLong(ctx, client, "kube-controller-manager")).To(BeTrue(),
				"Failed to wait for kube-controller-manager Cluster Operator to become available")

			By("waiting for cluster-api cluster operator to become available")
			Expect(framework.WaitForStatusAvailableMedium(ctx, client, "cluster-api")).To(BeTrue(),
				"Failed to wait for cluster-api Cluster Operator to become available")

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

			By("removing the mitm-proxy")
			framework.DeleteProxy(client)
		})
	})
