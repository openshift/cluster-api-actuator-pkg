package operators

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var _ = Describe("Cluster autoscaler operator should", framework.LabelAutoscaler, func() {
	var (
		client   runtimeclient.Client
		ctx      context.Context
		gatherer *gatherer.StateGatherer
	)

	BeforeEach(func() {
		var err error

		ctx = framework.GetContext()

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ok := framework.WaitForValidatingWebhook(ctx, client, "autoscaling.openshift.io")
		Expect(ok).To(BeTrue(), "Failed to wait for ValidatingWebhook")
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
		}
	})

	It("reject invalid ClusterAutoscaler resources early via webhook", func() {
		invalidCA := &caov1.ClusterAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterAutoscaler",
				APIVersion: "autoscaling.openshift.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				// Only "default" is allowed.
				Name: "invalid-name",
			},
		}

		Expect(client.Create(context.TODO(), invalidCA)).ToNot(Succeed(), "Failed to create invalid ClusterAutoscaler")
	})

	It("reject invalid MachineAutoscaler resources early via webhook", func() {
		invalidMA := &caov1beta1.MachineAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MachineAutoscaler",
				APIVersion: "autoscaling.openshift.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-%d", time.Now().Unix()),
				Namespace: framework.MachineAPINamespace,
			},
			Spec: caov1beta1.MachineAutoscalerSpec{
				// Min is greater than max, which is invalid.
				MinReplicas: 8,
				MaxReplicas: 2,
				ScaleTargetRef: caov1beta1.CrossVersionObjectReference{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
					Name:       "test",
				},
			},
		}

		Expect(client.Create(context.TODO(), invalidMA)).ToNot(Succeed(), "Failed to create invalid MachineAutoscaler")
	})
})

var _ = Describe("Cluster autoscaler operator deployment should", framework.LabelAutoscaler, framework.LabelLEVEL0, func() {
	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()
		Expect(framework.IsDeploymentAvailable(ctx, client, "cluster-autoscaler-operator", framework.MachineAPINamespace)).To(BeTrue(),
			"Failed to wait for cluster-autoscaler-operator Deployment to be available")
	})
})

var _ = Describe("Cluster autoscaler cluster operator status should", framework.LabelAutoscaler, framework.LabelLEVEL0, func() {
	It("be available", func() {
		client, err := framework.LoadClient()
		Expect(err).NotTo(HaveOccurred(), "Failed to load client")

		ctx := framework.GetContext()

		Expect(framework.WaitForStatusAvailableShort(ctx, client, "cluster-autoscaler")).To(BeTrue(),
			"Failed to wait for cluster-autoscaler Cluster Operator to be available")
	})
})

const (
	caoOperatorDeployment = "cluster-autoscaler-operator"
	caoWebhookServiceName = "cluster-autoscaler-operator"
	apiServerName         = "cluster"
	featureGateName       = "cluster"
)

// customTLSProfile returns a Custom TLS profile with a restricted cipher suite set for testing.
func customTLSProfile() *configv1.TLSSecurityProfile {
	return &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				MinTLSVersion: configv1.VersionTLS12,
				Ciphers: []string{
					"ECDHE-ECDSA-AES128-GCM-SHA256",
					"ECDHE-RSA-AES128-GCM-SHA256",
				},
			},
		},
	}
}

func isFeatureGateEnabled(ctx context.Context, client runtimeclient.Client, name configv1.FeatureGateName) bool {
	fg := &configv1.FeatureGate{}
	key := types.NamespacedName{Name: featureGateName}

	if err := client.Get(ctx, key, fg); err != nil {
		return false
	}

	for _, details := range fg.Status.FeatureGates {
		for _, enabled := range details.Enabled {
			if enabled.Name == name {
				return true
			}
		}
	}

	return false
}

func getAPIServer(ctx context.Context, client runtimeclient.Client) *configv1.APIServer {
	apiServer := &configv1.APIServer{}
	key := types.NamespacedName{Name: apiServerName}
	ExpectWithOffset(1, client.Get(ctx, key, apiServer)).To(Succeed(), "Failed to get APIServer")

	return apiServer
}

func patchAPIServerTLSAdherence(ctx context.Context, client runtimeclient.Client, adherence configv1.TLSAdherencePolicy) {
	apiServer := getAPIServer(ctx, client)
	patch := runtimeclient.MergeFrom(apiServer.DeepCopy())
	apiServer.Spec.TLSAdherence = adherence
	ExpectWithOffset(1, client.Patch(ctx, apiServer, patch)).To(Succeed(), "Failed to patch APIServer TLSAdherence")
}

func patchAPIServerTLSProfile(ctx context.Context, client runtimeclient.Client, profile *configv1.TLSSecurityProfile) {
	apiServer := getAPIServer(ctx, client)
	patch := runtimeclient.MergeFrom(apiServer.DeepCopy())
	apiServer.Spec.TLSSecurityProfile = profile
	ExpectWithOffset(1, client.Patch(ctx, apiServer, patch)).To(Succeed(), "Failed to patch APIServer TLSSecurityProfile")
}

func restoreAPIServerTLS(ctx context.Context, client runtimeclient.Client, original *configv1.APIServer) {
	apiServer := getAPIServer(ctx, client)
	patch := runtimeclient.MergeFrom(apiServer.DeepCopy())
	apiServer.Spec.TLSSecurityProfile = original.Spec.TLSSecurityProfile

	// TLSAdherence cannot be removed once set, so fall back to
	// LegacyAdheringComponentsOnly if the original had no value.
	adherence := original.Spec.TLSAdherence
	if adherence == "" {
		adherence = configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly
	}

	apiServer.Spec.TLSAdherence = adherence
	ExpectWithOffset(1, client.Patch(ctx, apiServer, patch)).To(Succeed(), "Failed to restore APIServer TLS config")
}

// OpenSSL cipher suite names used by curl's --ciphers flag. Make sure these are or are not in customTLSProfile(), as appropriate.
const (
	// cipherSuiteDisallowed is a cipher suite NOT in the custom TLS profile.
	cipherSuiteDisallowed = "ECDHE-RSA-AES256-GCM-SHA384"
	// cipherSuiteAllowed is a cipher suite that IS in the custom TLS profile.
	cipherSuiteAllowed = "ECDHE-RSA-AES128-GCM-SHA256"
)

const (
	tlsProbeDeploymentName = "tls-probe"
	tlsProbeAppLabel       = "tls-probe"
)

// tlsProbe manages a long-lived Deployment inside the cluster for running TLS handshake
// tests via oc exec. A Deployment is used instead of a bare Pod so the probe survives
// node drains triggered by APIServer config changes.
type tlsProbe struct {
	clientset   *kubernetes.Clientset
	webhookAddr string // resolved from the service during setup, e.g. "https://host:port"
}

// setup resolves the webhook service address and creates a single-replica Deployment
// that can be used to exec curl commands.
func (p *tlsProbe) setup(ctx context.Context) {
	svc, err := p.clientset.CoreV1().Services(framework.MachineAPINamespace).Get(ctx, caoWebhookServiceName, metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to get CAO webhook service")

	var port int32

	for _, sp := range svc.Spec.Ports {
		if sp.Name == "https" {
			port = sp.Port
			break
		}
	}

	ExpectWithOffset(1, port).NotTo(BeZero(), "webhook service %s has no 'https' port", caoWebhookServiceName)

	p.webhookAddr = fmt.Sprintf("https://%s.%s.svc:%d", caoWebhookServiceName, framework.MachineAPINamespace, port)

	probeLabels := map[string]string{"app": tlsProbeAppLabel}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsProbeDeploymentName,
			Namespace: framework.MachineAPINamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: probeLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: probeLabels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "probe",
							Image:   "registry.access.redhat.com/ubi9-minimal:latest",
							Command: []string{"sleep", "infinity"},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"true"},
									},
								},
								PeriodSeconds: 5,
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = p.clientset.AppsV1().Deployments(framework.MachineAPINamespace).Create(ctx, deploy, metav1.CreateOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to create TLS probe deployment")

	p.waitForReady(ctx, 120*time.Second)
}

// teardown deletes the probe Deployment and waits for it to be fully removed.
func (p *tlsProbe) teardown(ctx context.Context) {
	propagation := metav1.DeletePropagationForeground

	err := p.clientset.AppsV1().Deployments(framework.MachineAPINamespace).Delete(ctx, tlsProbeDeploymentName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		return
	}

	Eventually(func() error {
		_, err := p.clientset.AppsV1().Deployments(framework.MachineAPINamespace).Get(ctx, tlsProbeDeploymentName, metav1.GetOptions{})
		return err
	}, 60*time.Second, 2*time.Second).ShouldNot(Succeed())
}

// waitForReady polls until the probe Deployment has at least one ready pod.
func (p *tlsProbe) waitForReady(ctx context.Context, timeout time.Duration) {
	Eventually(func() int32 {
		deploy, err := p.clientset.AppsV1().Deployments(framework.MachineAPINamespace).Get(ctx, tlsProbeDeploymentName, metav1.GetOptions{})
		if err != nil {
			return 0
		}

		return deploy.Status.ReadyReplicas
	}, timeout, 2*time.Second).Should(BeNumerically(">", int32(0)), "TLS probe deployment never became ready")
}

// readyPodName returns the name of a Ready pod owned by the probe Deployment.
// It waits up to the given timeout for a ready pod to appear (handles node drain recovery).
func (p *tlsProbe) readyPodName(ctx context.Context, timeout time.Duration) string {
	selector := labels.SelectorFromSet(labels.Set{"app": tlsProbeAppLabel})

	var podName string

	Eventually(func() bool {
		podList, err := p.clientset.CoreV1().Pods(framework.MachineAPINamespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return false
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			if pod.DeletionTimestamp != nil {
				continue
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					podName = pod.Name
					return true
				}
			}
		}

		return false
	}, timeout, 2*time.Second).Should(BeTrue(), "no ready TLS probe pod found")

	return podName
}

// Curl exit codes for TLS negotiation failures.
const (
	// curlExitSSLConnectError is curl exit code 35: SSL connect error.
	curlExitSSLConnectError = 35
	// curlExitSSLCipherError is curl exit code 59: no cipher suites in common.
	curlExitSSLCipherError = 59
)

// kubeconfigPath returns the kubeconfig path from the -kubeconfig flag or KUBECONFIG env var.
func kubeconfigPath() string {
	if f := flag.Lookup("kubeconfig"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}

	return os.Getenv("KUBECONFIG")
}

// handshake runs a curl command in a ready probe pod to test a TLS handshake with the
// CAO webhook using the specified cipher suite. Returns nil if the handshake succeeded.
// It first waits for a ready pod from the probe Deployment to handle node drain recovery.
func (p *tlsProbe) handshake(ctx context.Context, cipherSuite string) error {
	podName := p.readyPodName(ctx, 120*time.Second)

	args := []string{}

	if kc := kubeconfigPath(); kc != "" {
		args = append(args, "--kubeconfig", kc)
	}

	args = append(args,
		"exec", podName,
		"-n", framework.MachineAPINamespace,
		"--", "curl",
		"--insecure",
		"--ciphers", cipherSuite,
		"--tls-max", "1.2",
		"--tlsv1.2",
		"--connect-timeout", "10",
		"--max-time", "15",
		"--silent",
		"--output", "/dev/null",
		p.webhookAddr,
	)

	cmd := exec.CommandContext(ctx, "oc", args...) //nolint:gosec // test-only: arguments are constructed from constants

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("TLS handshake failed with cipher suite %s: %w (stderr: %s)", cipherSuite, err, stderr.String())
	}

	return nil
}

// isTLSNegotiationError returns true if the error is from a TLS/cipher negotiation
// failure (curl exit codes 35 or 59), as opposed to connectivity or other errors.
func isTLSNegotiationError(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}

	code := exitErr.ExitCode()

	return code == curlExitSSLConnectError || code == curlExitSSLCipherError
}

// expectHandshakeRejected returns a function suitable for Eventually that
// succeeds only when the TLS handshake with the given cipher suite fails
// due to a TLS negotiation error (not a generic connectivity failure).
func expectHandshakeRejected(probe *tlsProbe, ctx context.Context, cipherSuite string) func() error {
	return func() error {
		err := probe.handshake(ctx, cipherSuite)
		if err == nil {
			return fmt.Errorf("expected TLS handshake to fail with cipher suite %s, but it succeeded", cipherSuite)
		}

		if isTLSNegotiationError(err) {
			return nil
		}

		return fmt.Errorf("handshake failed for non-TLS reason (expected curl exit 35 or 59): %w", err)
	}
}

var _ = Describe("Cluster autoscaler operator TLS adherence",
	framework.LabelAutoscaler, framework.LabelPeriodic, framework.LabelDisruptive, Ordered, ContinueOnFailure, Serial, func() {
		var (
			client            runtimeclient.Client
			ctx               context.Context
			g                 *gatherer.StateGatherer
			originalAPIServer *configv1.APIServer
			probe             *tlsProbe
		)

		BeforeAll(func() {
			var err error

			ctx = framework.GetContext()

			client, err = framework.LoadClient()
			Expect(err).NotTo(HaveOccurred(), "Failed to load client")

			if !isFeatureGateEnabled(ctx, client, "TLSAdherence") {
				Skip("TLSAdherence feature gate is not enabled, skipping TLS adherence tests")
			}

			By("Creating TLS probe deployment")

			clientset, err := framework.LoadClientset()
			Expect(err).NotTo(HaveOccurred(), "Failed to load clientset")

			probe = &tlsProbe{clientset: clientset}
			probe.setup(ctx)
		})

		BeforeEach(func() {
			var err error

			g, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

			By("Saving original APIServer TLS configuration")

			originalAPIServer = getAPIServer(ctx, client).DeepCopy()

			By("Verifying CAO operator deployment is available")
			Expect(framework.IsDeploymentAvailable(ctx, client, caoOperatorDeployment, framework.MachineAPINamespace)).To(BeTrue(),
				"cluster-autoscaler-operator deployment is not available")
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() && g != nil {
				Expect(g.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to GatherAll")
			}

			if originalAPIServer != nil {
				By("Restoring original APIServer TLS configuration")
				restoreAPIServerTLS(ctx, client, originalAPIServer)

				By("Waiting for CAO operator to reconcile after restoring APIServer config")
				Eventually(func() bool {
					return framework.IsDeploymentAvailable(ctx, client, caoOperatorDeployment, framework.MachineAPINamespace)
				}, framework.WaitOverMedium, framework.RetryMedium).Should(BeTrue(),
					"cluster-autoscaler-operator deployment did not become available after restoring APIServer config")
			}
		})

		AfterAll(func() {
			if probe != nil {
				By("Cleaning up TLS probe deployment")
				probe.teardown(ctx)
			}

			if client == nil {
				return
			}

			By("Waiting for cluster operators to stabilize after TLS configuration changes")

			if !framework.WaitForStatusAvailableOverLong(ctx, client, "kube-apiserver") {
				GinkgoWriter.Printf("WARNING: kube-apiserver ClusterOperator did not become available\n")
			}

			if !framework.WaitForStatusAvailableOverLong(ctx, client, "openshift-apiserver") {
				GinkgoWriter.Printf("WARNING: openshift-apiserver ClusterOperator did not become available\n")
			}

			if !framework.WaitForStatusAvailableMedium(ctx, client, "machine-api") {
				GinkgoWriter.Printf("WARNING: machine-api ClusterOperator did not become available\n")
			}

			coList := &configv1.ClusterOperatorList{}
			if err := client.List(ctx, coList); err != nil {
				GinkgoWriter.Printf("WARNING: failed to list ClusterOperators: %v\n", err)
				return
			}

			for _, co := range coList.Items {
				if !framework.WaitForStatusAvailableOverLong(ctx, client, co.Name) {
					GinkgoWriter.Printf("WARNING: %s ClusterOperator did not become available\n", co.Name)
				}
			}
		})

		It("should configure webhook TLS to match custom cluster profile when TLSAdherence is StrictAllComponents", func() {
			By("Setting APIServer TLSSecurityProfile to a custom profile with restricted cipher suites")
			patchAPIServerTLSProfile(ctx, client, customTLSProfile())

			By("Setting APIServer TLSAdherence to StrictAllComponents")
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyStrictAllComponents)

			By("Verifying webhook rejects connections with cipher suites outside the custom profile")
			Eventually(expectHandshakeRejected(probe, ctx, cipherSuiteDisallowed), framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())

			By("Verifying webhook accepts connections with cipher suites in the custom profile")
			Eventually(func() error {
				return probe.handshake(ctx, cipherSuiteAllowed)
			}, framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())
		})

		It("should use default TLS config when TLSAdherence is LegacyAdheringComponentsOnly", func() {
			By("Setting APIServer TLSSecurityProfile to a custom profile with restricted cipher suites")
			patchAPIServerTLSProfile(ctx, client, customTLSProfile())

			By("Setting APIServer TLSAdherence to LegacyAdheringComponentsOnly")
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly)

			By("Verifying webhook accepts connections with cipher suites outside the custom profile")
			Eventually(func() error {
				return probe.handshake(ctx, cipherSuiteDisallowed)
			}, framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())
		})

		It("should update webhook TLS when TLSAdherence transitions to StrictAllComponents", func() {
			By("Setting TLSAdherence to LegacyAdheringComponentsOnly")
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly)

			By("Verifying webhook accepts the broader default cipher suite set")
			Eventually(func() error {
				return probe.handshake(ctx, cipherSuiteDisallowed)
			}, framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())

			By("Setting TLSSecurityProfile to custom and TLSAdherence to StrictAllComponents")
			patchAPIServerTLSProfile(ctx, client, customTLSProfile())
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyStrictAllComponents)

			By("Verifying webhook now rejects connections with cipher suites outside the custom profile")
			Eventually(expectHandshakeRejected(probe, ctx, cipherSuiteDisallowed), framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())
		})

		It("should revert webhook TLS when TLSAdherence transitions from StrictAllComponents to LegacyAdheringComponentsOnly", func() {
			By("Setting TLSSecurityProfile to custom and TLSAdherence to StrictAllComponents")
			patchAPIServerTLSProfile(ctx, client, customTLSProfile())
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyStrictAllComponents)

			By("Verifying webhook only accepts the custom cipher suites")
			Eventually(expectHandshakeRejected(probe, ctx, cipherSuiteDisallowed), framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())

			By("Setting TLSAdherence to LegacyAdheringComponentsOnly")
			patchAPIServerTLSAdherence(ctx, client, configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly)

			By("Verifying webhook now accepts the broader default cipher suite set again")
			Eventually(func() error {
				return probe.handshake(ctx, cipherSuiteDisallowed)
			}, framework.WaitOverMedium, framework.RetryMedium).Should(Succeed())
		})
	})
