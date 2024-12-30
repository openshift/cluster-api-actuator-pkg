package annotations

import (
	"context"
	"fmt"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	namespace = "openshift-machine-api"

	// Timeout constants for service operations.
	serviceTimeout      = 3 * time.Minute
	servicePollInterval = 10 * time.Second
)

var cl client.Client

var _ = Describe("Service Annotation tests GCP", framework.LabelCCM, framework.LabelDisruptive, Ordered, func() {
	var (
		ctx             context.Context
		platform        configv1.PlatformType
		createdServices []string
		gcpClient       *framework.GCPClient
	)

	BeforeAll(func() {
		cfg, err := config.GetConfig()
		Expect(err).ToNot(HaveOccurred(), "Failed to GetConfig")

		cl, err = client.New(cfg, client.Options{})
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(cl)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, cl)
		fmt.Println("platform is ", platform)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")
		if platform != configv1.GCPPlatformType {
			Skip("Skipping GCP E2E tests")
		}

		// Get GCP credentials from infrastructure
		project, region, err := framework.GetGCPCredentialsFromInfrastructure(ctx, cl)
		Expect(err).ToNot(HaveOccurred(), "Failed to get GCP credentials from infrastructure")
		gcpClient = framework.NewGCPClient(project, region)
	})

	AfterAll(func() {
		for _, svcName := range createdServices {
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      svcName,
					Namespace: namespace,
				},
			}
			_ = cl.Delete(ctx, service)
		}
	})

	It("should validate network-tier annotation with GCP CLI verification", func() {
		serviceName := "test-service-network-tier-validation"

		// Step 1: Create service with network-tier: Standard
		By("Creating service with network-tier: Standard")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
				Annotations: map[string]string{
					"cloud.google.com/network-tier": "Standard",
				},
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{"app": "test"},
				Ports: []corev1.ServicePort{{
					Port: 80,
				}},
			},
		}
		Expect(cl.Create(ctx, service)).To(Succeed())
		createdServices = append(createdServices, service.Name)

		// Wait for service to get an external IP
		var ingressIP string
		Eventually(func() (string, error) {
			updatedService := &corev1.Service{}
			err := cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)
			if err != nil {
				return "", fmt.Errorf("failed to get updated service: %w", err)
			}
			if len(updatedService.Status.LoadBalancer.Ingress) > 0 {
				ingressIP = updatedService.Status.LoadBalancer.Ingress[0].IP

				return ingressIP, nil
			}

			return "", nil
		}, serviceTimeout, servicePollInterval).ShouldNot(BeEmpty(), "LoadBalancer service did not get an external IP")

		// Verify with GCP CLI that LB has network-tier: Standard
		By("Verifying with GCP CLI that load balancer has network-tier: Standard")
		Eventually(func() error {
			return gcpClient.WaitForLoadBalancerNetworkTier(ctx, ingressIP, "STANDARD", 2*time.Minute)
		}, serviceTimeout, servicePollInterval).Should(Succeed(), "Load balancer should have STANDARD network tier")

		// Step 2: Update service annotation to network-tier: Premium
		By("Updating service annotation to network-tier: Premium")
		latestService := &corev1.Service{}
		Expect(cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, latestService)).To(Succeed())
		latestService.Annotations["cloud.google.com/network-tier"] = "Premium"
		Expect(cl.Update(ctx, latestService)).To(Succeed())

		// Wait for IP to change (network tier change requires new IP)
		var newIngressIP string
		Eventually(func() (string, error) {
			updatedService := &corev1.Service{}
			err := cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)
			if err != nil {
				return "", fmt.Errorf("failed to get updated service: %w", err)
			}
			if len(updatedService.Status.LoadBalancer.Ingress) > 0 {
				currentIP := updatedService.Status.LoadBalancer.Ingress[0].IP
				klog.Infof("Polling for IP change - Current IP: %s, Original IP: %s", currentIP, ingressIP)
				if currentIP != ingressIP && currentIP != "" {
					klog.Infof("IP successfully changed from %s to %s", ingressIP, currentIP)
					newIngressIP = currentIP

					return currentIP, nil
				}
				// IP hasn't changed yet, continue polling
				klog.Infof("IP has not changed yet, continuing to wait...")
			}

			return "", nil
		}, serviceTimeout, servicePollInterval).ShouldNot(BeEmpty(), "IP should change after network tier update within 4 minutes")

		// Ensure newIngressIP is not empty before proceeding
		Expect(newIngressIP).NotTo(BeEmpty(), "New ingress IP should not be empty")
		klog.Infof("Proceeding with new IP: %s", newIngressIP)

		// Verify with GCP CLI that LB has network-tier: Premium
		By("Verifying with GCP CLI that load balancer has network-tier: Premium")
		Eventually(func() error {
			return gcpClient.WaitForLoadBalancerNetworkTier(ctx, newIngressIP, "Premium", 2*time.Minute)
		}, serviceTimeout, servicePollInterval).Should(Succeed(), "Load balancer should have PREMIUM network tier")

		// Step 3: Test that empty string value is rejected by validation policy
		By("Testing that empty string value is rejected by validation policy")
		latestService = &corev1.Service{}
		Expect(cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, latestService)).To(Succeed())
		latestService.Annotations["cloud.google.com/network-tier"] = ""

		// The update should fail due to validation policy rejecting empty string
		err := cl.Update(ctx, latestService)
		Expect(err).To(HaveOccurred(), "Service update should fail with empty string network-tier value")
		Expect(err.Error()).To(ContainSubstring("must be either 'Standard' or 'Premium'"),
			"Error should indicate that only Standard or Premium values are allowed")

		// Verify the service still has the previous Premium annotation
		updatedService := &corev1.Service{}
		Expect(cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)).To(Succeed())
		Expect(updatedService.Annotations["cloud.google.com/network-tier"]).To(Equal("Premium"),
			"Service should retain Premium annotation after failed update")

	})

	It("should reject invalid network-tier annotation values", func() {
		serviceName := "test-service-invalid-network-tier"

		// Step 4: Create service with invalid network-tier value
		By("Creating service with invalid network-tier: InvalidValue")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
				Annotations: map[string]string{
					"cloud.google.com/network-tier": "InvalidValue",
				},
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{"app": "test"},
				Ports: []corev1.ServicePort{{
					Port: 80,
				}},
			},
		}

		// The creation should fail due to invalid annotation value
		err := cl.Create(ctx, service)
		Expect(err).To(HaveOccurred(), "Service creation should fail with invalid network-tier value")

		// Verify the service was not created
		createdService := &corev1.Service{}
		err = cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, createdService)
		Expect(err).To(HaveOccurred(), "Service should not exist after failed creation")
	})

	It("should validate other GCP annotations(local-with-fallback, load-balancer-backend-share, internal-load-balancer-allow-global-access, internal-load-balancer-subnet)", func() {
		serviceName := "test-service-other-gcp-annotations"

		// Test other GCP annotations that don't require CLI verification
		annotationsToTest := map[string]string{
			"traffic-policy.network.alpha.openshift.io/local-with-fallback": "true",
			"alpha.cloud.google.com/load-balancer-backend-share":            "",
			"networkingke.io/internal-load-balancer-allow-global-access":    "true",
			"networkingke.io/internal-load-balancer-subnet":                 "",
		}

		By("Creating service with other GCP annotations")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName,
				Namespace:   namespace,
				Annotations: annotationsToTest,
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{"app": "test"},
				Ports: []corev1.ServicePort{{
					Port: 80,
				}},
			},
		}
		Expect(cl.Create(ctx, service)).To(Succeed())
		createdServices = append(createdServices, service.Name)

		// Wait for service to get an external IP
		Eventually(func() (string, error) {
			updatedService := &corev1.Service{}
			err := cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)
			if err != nil {
				return "", fmt.Errorf("failed to get updated service: %w", err)
			}
			if len(updatedService.Status.LoadBalancer.Ingress) > 0 {

				return updatedService.Status.LoadBalancer.Ingress[0].IP, nil
			}

			return "", nil
		}, serviceTimeout, servicePollInterval).ShouldNot(BeEmpty(), "LoadBalancer service did not get an external IP")

		// Verify annotations are preserved
		updatedService := &corev1.Service{}
		Expect(cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)).To(Succeed())
		for key, value := range annotationsToTest {
			Expect(updatedService.Annotations).To(HaveKeyWithValue(key, value),
				fmt.Sprintf("Annotation %s should be preserved", key))
		}
	})
})
