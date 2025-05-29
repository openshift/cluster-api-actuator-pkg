package annotations

import (
	"context"
	"fmt"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
)

var (
	annotationsToTest = map[string][]string{
		"traffic-policy.network.alpha.openshift.io/local-with-fallback": {"true"},
		"alpha.cloud.google.com/load-balancer-backend-share":            {""},
		"networking.gke.io/internal-load-balancer-allow-global-access":  {"true"},
		"networking.gke.io/internal-load-balancer-subnet":               {""},
		"cloud.google.com/network-tier":                                 {"Standard", "Premium", "InvalidValue"},
	}
)

const (
	namespace = "openshift-machine-api"
)

var cl client.Client

var _ = g.Describe("Service Annotation tests GCP", framework.LabelCCM, framework.LabelDisruptive, g.Ordered, func() {
	var (
		ctx             context.Context
		platform        configv1.PlatformType
		createdServices []string
	)

	g.BeforeAll(func() {
		cfg, err := config.GetConfig()
		o.Expect(err).ToNot(o.HaveOccurred(), "Failed to GetConfig")

		cl, err = client.New(cfg, client.Options{})
		o.Expect(err).NotTo(o.HaveOccurred(), "Failed to create Kubernetes client for test")
		komega.SetClient(cl)
		ctx = framework.GetContext()
		platform, err = framework.GetPlatform(ctx, cl)
		fmt.Println("platform is ", platform)
		o.Expect(err).ToNot(o.HaveOccurred(), "Failed to get platform")
		if platform != configv1.GCPPlatformType {
			g.Skip("Skipping GCP E2E tests")
		}

	})

	g.AfterAll(func() {
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

	g.It("should validate annotations including network-tier and IP changes", func() {
		g.By("Create service without annotations")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service-annotation-validation",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{"app": "test"},
				Ports: []corev1.ServicePort{{
					Port: 80,
				}},
			},
		}
		o.Expect(cl.Create(ctx, service)).To(o.Succeed())
		createdServices = append(createdServices, service.Name)

		var lastIngressIP string
		o.Eventually(func() (string, error) {
			updatedService := &corev1.Service{}
			err := cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)
			if err != nil {
				return "", fmt.Errorf("failed to get updated service: %w", err)
			}
			if len(updatedService.Status.LoadBalancer.Ingress) > 0 {
				lastIngressIP = updatedService.Status.LoadBalancer.Ingress[0].IP
				return lastIngressIP, nil
			}

			return "", nil
		}, 2*time.Minute, 10*time.Second).ShouldNot(o.BeEmpty(), "LoadBalancer service did not get an external IP")

		for key, values := range annotationsToTest {
			for _, value := range values {
				g.By(fmt.Sprintf("Adding annotation: %s=%s", key, value))
				latestService := &corev1.Service{}
				o.Expect(cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, latestService)).To(o.Succeed())

				if latestService.Annotations == nil {
					latestService.Annotations = make(map[string]string)
				}
				latestService.Annotations[key] = value

				if key == "cloud.google.com/network-tier" && value != "Standard" && value != "Premium" {
					o.Expect(cl.Update(ctx, latestService)).ToNot(o.Succeed(), "The annotation 'cloud.google.com/network-tier', if specified, must be either 'Standard' or 'Premium'")
					continue
				}

				o.Expect(cl.Update(ctx, latestService)).To(o.Succeed())

				if key == "cloud.google.com/network-tier" {
					g.By(fmt.Sprintf("Validating Ingress IP change after annotation update: %s=%s", key, value))
					o.Eventually(func() (string, error) {
						updatedService := &corev1.Service{}
						err := cl.Get(ctx, client.ObjectKey{Name: service.Name, Namespace: namespace}, updatedService)
						if err != nil {
							return "", fmt.Errorf("failed to get updated service: %w", err)
						}
						if len(updatedService.Status.LoadBalancer.Ingress) > 0 {
							return updatedService.Status.LoadBalancer.Ingress[0].IP, nil
						}

						return "", nil
					}, 4*time.Minute, 10*time.Second).ShouldNot(o.Equal(lastIngressIP), "Ingress IP did not change after annotation update")
				}
			}
		}
	})
})
