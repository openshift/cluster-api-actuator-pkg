package operators

import (
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	discoveryv1 "k8s.io/api/discovery/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
)

const (
	cmaDeployment      = "machine-approver"
	cmaClusterOperator = "machine-approver"
	cmaNamespace       = "openshift-cluster-machine-approver"
	cmaDeploymentcapi  = "machine-approver-capi"
	cmaServicecapi     = "machine-approver-capi"
	cmaServicePortcapi = 9194
	cmaService         = "machine-approver"
	cmaServicePort     = 9192
	cmaMetricsPort     = 9193
)

var _ = Describe("Cluster Machine Approver deployment", framework.LabelMachineApprover, framework.LabelLEVEL0, func() {
	It("should be available", func() {
		ctx := framework.GetContext()

		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		Expect(framework.IsDeploymentAvailable(ctx, client, cmaDeployment, cmaNamespace)).To(BeTrue(),
			"Failed to wait for cluster-machine-approver Deployment to become available")
	})
})

var _ = Describe("Cluster Machine Approver Cluster Operator Status", framework.LabelMachineApprover, framework.LabelLEVEL0, func() {
	It("should be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()

		Expect(framework.WaitForStatusAvailableShort(ctx, client, cmaClusterOperator)).To(BeTrue(),
			"Failed to wait for cluster-machine-approver Cluster Operator to be available")
	})
})

var _ = Describe("Cluster Machine Approver CAPI Integration", framework.LabelMachineApprover, framework.LabelLEVEL0, func() {
	It("cluster-machine-approver must have endpoint slices for open ports the operator uses", func() {
		ctx := framework.GetContext()
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		// Pre-check: Skip if cluster doesn't have TechPreviewNoUpgrade or CustomNoUpgrade featuregate enabled
		oc, err := framework.NewCLI()
		Expect(err).NotTo(HaveOccurred(), "Failed to create CLI")

		featureSet, err := oc.WithoutNamespace().Run("get").Args("featuregate", "cluster", "-o=jsonpath={.spec.featureSet}").Output()
		Expect(err).NotTo(HaveOccurred(), "Failed to get featureSet")

		if featureSet != "TechPreviewNoUpgrade" && featureSet != "CustomNoUpgrade" {
			Skip("Cluster does not have TechPreviewNoUpgrade or CustomNoUpgrade featuregate enabled. Not a valid cluster for this test.")
		}

		// Test Case 1: Verify machine-approver-capi deployment has port 9193 configured with internal IP 127.0.0.1
		deployment, err := framework.GetDeployment(ctx, client, cmaDeploymentcapi, cmaNamespace)
		Expect(err).NotTo(HaveOccurred(), "Failed to get machine-approver-capi deployment")

		// Check containers for the port configuration
		foundInternalPort := false
		foundUpstreamArg := false
		for _, container := range deployment.Spec.Template.Spec.Containers {
			// Check for upstream argument with 127.0.0.1:9193
			for _, arg := range container.Args {
				if strings.Contains(arg, "--upstream=") && strings.Contains(arg, "127.0.0.1:"+strconv.Itoa(cmaMetricsPort)) {
					foundUpstreamArg = true
				}
			}

			// Check for environment variable with port 9193
			for _, env := range container.Env {
				if env.Value == "9193" || strings.Contains(env.Value, "9193") {
					foundInternalPort = true
				}
			}
		}

		Expect(foundUpstreamArg && foundInternalPort).To(BeTrue(),
			"Deployment machine-approver-capi should have port 9193 configured with 127.0.0.1 (internal)")

		// Test Case 2: Verify services exist at ports 9192 and 9194
		// Check machine-approver service (port 9192)
		maSvc, err := framework.GetService(ctx, client, cmaService, cmaNamespace)
		Expect(err).NotTo(HaveOccurred(), "Failed to get machine-approver service")

		foundMaPort := false
		for _, port := range maSvc.Spec.Ports {
			if port.Port == cmaServicePort {
				foundMaPort = true
				break
			}
		}
		Expect(foundMaPort).To(BeTrue(),
			"Service machine-approver should have port 9192")

		// Check machine-approver-capi service (port 9194)
		maCapiSvc, err := framework.GetService(ctx, client, cmaServicecapi, cmaNamespace)
		Expect(err).NotTo(HaveOccurred(), "Failed to get machine-approver-capi service")

		foundCapiPort := false
		for _, port := range maCapiSvc.Spec.Ports {
			if port.Port == cmaServicePortcapi {
				foundCapiPort = true
				break
			}
		}
		Expect(foundCapiPort).To(BeTrue(),
			"Service machine-approver-capi should have port 9194")

		// Test Case 3: Verify endpoint slices exist for above ports
		endpointSlices := &discoveryv1.EndpointSliceList{}
		err = client.List(ctx, endpointSlices, runtimeclient.InNamespace(cmaNamespace))
		Expect(err).NotTo(HaveOccurred(), "Failed to list endpoint slices")

		// Find endpoint slices for machine-approver (port 9192)
		foundMaEndpoint := false
		foundCapiEndpoint := false

		for _, slice := range endpointSlices.Items {
			serviceName := slice.Labels["kubernetes.io/service-name"]

			// Check for machine-approver endpoint slice with port 9192
			if serviceName == cmaService {
				for _, port := range slice.Ports {
					if port.Port != nil && *port.Port == cmaServicePort {
						foundMaEndpoint = true
						break
					}
				}
			}

			// Check for machine-approver-capi endpoint slice with port 9194
			if serviceName == cmaServicecapi {
				for _, port := range slice.Ports {
					if port.Port != nil && *port.Port == cmaServicePortcapi {
						foundCapiEndpoint = true
						break
					}
				}
			}
		}

		Expect(foundMaEndpoint).To(BeTrue(),
			"EndpointSlice for machine-approver service (port 9192) should exist")
		Expect(foundCapiEndpoint).To(BeTrue(),
			"EndpointSlice for machine-approver-capi service (port 9194) should exist")
	})
})
