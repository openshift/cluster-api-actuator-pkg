/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	configv1 "github.com/openshift/api/config/v1"
	appsv1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/apps/v1"
	corev1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/core/v1"
)

const (
	proxyNamespace         = MachineAPINamespace
	proxyName              = "mitm-proxy"
	mitmSignerName         = "mitm-signer"
	mitmBootstrapName      = "mitm-bootstrap"
	mitmCustomPKIName      = "mitm-custom-pki"
	mitmCustomPKINamespace = "openshift-config"
	mitmDaemonsetName      = proxyName
	mitmServiceName        = proxyName
)

const proxySetup = `
cd /.mitmproxy
cat /root/certs/tls.key /root/certs/tls.crt > /.mitmproxy/mitmproxy-ca.pem
curl -O https://snapshots.mitmproxy.org/5.3.0/mitmproxy-5.3.0-linux.tar.gz
tar xvf mitmproxy-5.3.0-linux.tar.gz
HOME=/.mitmproxy ./mitmdump --ssl-insecure
`

// IsClusterProxyEnabled returns true if a global proxy is configured or still active in status.
func IsClusterProxyEnabled(ctx context.Context, c client.Client) (bool, error) {
	proxy := &configv1.Proxy{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, proxy); err != nil {
		return false, fmt.Errorf("failed to get cluster proxy: %w", err)
	}

	return proxy.Spec.HTTPProxy != "" || proxy.Spec.HTTPSProxy != "" ||
		proxy.Status.HTTPProxy != "" || proxy.Status.HTTPSProxy != "", nil
}

// GetConfiguredProxyStatus returns proxy status only after configured endpoints have propagated.
// Callers should retry while status is empty or differs from spec, including on preconfigured clusters.
func GetConfiguredProxyStatus(ctx context.Context, c client.Client) (configv1.ProxyStatus, error) {
	proxy := &configv1.Proxy{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, proxy); err != nil {
		return configv1.ProxyStatus{}, fmt.Errorf("failed to get cluster proxy: %w", err)
	}

	if proxy.Status.HTTPProxy == "" && proxy.Status.HTTPSProxy == "" {
		return configv1.ProxyStatus{}, fmt.Errorf("waiting for configured Proxy status")
	}

	if proxy.Status.HTTPProxy != proxy.Spec.HTTPProxy || proxy.Status.HTTPSProxy != proxy.Spec.HTTPSProxy {
		return configv1.ProxyStatus{}, fmt.Errorf("waiting for Proxy endpoints to propagate from spec to status")
	}

	return proxy.Status, nil
}

// DeployProxy deploys a MITM Proxy to the cluster.
func DeployProxy(c client.Client, gomegaArgs ...interface{}) {
	ctx := context.Background()
	kom := komega.New(c)

	proxyLabels := map[string]string{
		"app": proxyName,
	}

	// Generate an RSA private key and its corresponding X.509 certificate,
	// for the MITM proxy.
	certBytes, keyBytes, err := generateCert()
	Expect(err).NotTo(HaveOccurred(), "generating private key and certificate should not error.")

	mitmSignerKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	mitmSignerCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	mitmSignerSecret := corev1resourcebuilder.Secret().WithName(mitmSignerName).WithNamespace(proxyNamespace).WithLabels(proxyLabels).
		WithData(map[string][]byte{"tls.crt": mitmSignerCert, "tls.key": mitmSignerKey}).Build()

	mitmBootstrapConfigMap := corev1resourcebuilder.ConfigMap().WithName(mitmBootstrapName).WithNamespace(proxyNamespace).WithLabels(proxyLabels).
		WithData(map[string]string{"startup.sh": proxySetup}).Build()

	mitmCustomPkiConfigMap := corev1resourcebuilder.ConfigMap().WithName(mitmCustomPKIName).WithNamespace(mitmCustomPKINamespace).
		WithData(map[string]string{"ca-bundle.crt": string(mitmSignerCert)}).Build()

	mitmDaemonset := appsv1resourcebuilder.DaemonSet().WithName(proxyName).WithNamespace(proxyNamespace).WithLabels(proxyLabels).
		WithVolumes(buildDaemonSetVolumes()).WithContainers(buildDaemonSetContainers()).Build()

	mitmService := corev1resourcebuilder.Service().WithNamespace(proxyNamespace).WithName(proxyName).
		WithLabels(proxyLabels).WithSelector(proxyLabels).WithPorts(buildServicePorts()).Build()

	By("Creating the MITM proxy Secret")
	Eventually(c.Create(ctx, mitmSignerSecret)).Should(Succeed(), "timed out creating the MITM proxy Secret.")

	By("Creating the MITM proxy ConfigMaps")
	Eventually(c.Create(ctx, mitmBootstrapConfigMap)).Should(Succeed(), "timed out creating the MITM proxy Bootstrap ConfigMap.")
	Eventually(c.Create(ctx, mitmCustomPkiConfigMap)).Should(Succeed(), "timed out creating the MITM proxy Custom PKI ConfigMap.")

	By("Creating the MITM proxy DaemonSet")
	Eventually(c.Create(ctx, mitmDaemonset)).Should(Succeed(), "timed out creating the MITM proxy DaemonSet.")

	By("Waiting for the MITM proxy DaemonSet to be available")
	Eventually(kom.Object(mitmDaemonset), time.Minute*1).Should(
		HaveField("Status.NumberAvailable", Not(BeZero())),
		"timed out waiting for MITM proxy DaemonSet to be available.",
	)

	By("Creating the MITM proxy Service")
	Eventually(c.Create(ctx, mitmService)).Should(Succeed(), "timed out creating the MITM proxy Service.")

	By("Waiting for the MITM proxy Service to be available")
	Eventually(kom.Object(mitmService), time.Minute*1).Should(
		HaveField("Spec.ClusterIP", Not(Equal(""))),
		"timed out waiting for the MITM proxy Service to be available.",
	)
}

// ConfigureClusterWideProxy configures the Cluster-Wide Proxy to use the MITM Proxy.
func ConfigureClusterWideProxy(c client.Client, gomegaArgs ...interface{}) {
	ctx := context.Background()
	kom := komega.New(c)

	services := &corev1.ServiceList{}
	Eventually(c.List(ctx, services, client.MatchingLabels(map[string]string{"app": "mitm-proxy"}))).Should(Succeed(), "timed out listing Services for app=mitm-proxy.")

	proxy := &configv1.Proxy{}
	Eventually(c.Get(ctx, client.ObjectKey{Name: "cluster"}, proxy)).Should(Succeed(), "timed out getting Proxy named 'cluster.'")

	Eventually(kom.Update(proxy, func() {
		proxy.Spec.HTTPProxy = "http://" + services.Items[0].Spec.ClusterIP + ":8080"
		proxy.Spec.HTTPSProxy = "http://" + services.Items[0].Spec.ClusterIP + ":8080"
		proxy.Spec.NoProxy = ".org,.com,.net,quay.io,registry.redhat.io"
		proxy.Spec.TrustedCA = configv1.ConfigMapNameReference{
			Name: mitmCustomPKIName,
		}
	}), gomegaArgs...).Should(Succeed(), "cluster wide proxy set be able to be updated")
}

// WaitForMAPIControllersProxyEnvVars waits for the MAPI controller containers in a Deployment
// to match the configured Proxy status, including absence of unconfigured variables.
// Metrics sidecars do not receive proxy variables and are excluded.
func WaitForMAPIControllersProxyEnvVars(c client.Client, namespace, deploymentName string) {
	ctx := context.Background()

	By(fmt.Sprintf("Waiting for MAPI controller containers in %s/%s to reflect configured cluster-wide proxy", namespace, deploymentName))

	status := WaitForConfiguredProxyStatus(ctx, c)
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: deploymentName}}

	Eventually(komega.New(c).Object(deploy), WaitOverMedium).Should(
		WithTransform(mapiControllersProxyEnvVars, HaveEach(ConsistOf(proxyEnvVarsFromStatus(status)))),
		"Deployment %s/%s did not reflect Proxy status", namespace, deploymentName,
	)
}

func mapiControllersProxyEnvVars(deploy *appsv1.Deployment) (map[string][]corev1.EnvVar, error) {
	all := proxyEnvVarsByContainer(deploy)
	selected := map[string][]corev1.EnvVar{}

	for _, name := range []string{"machine-controller", "machineset-controller", "nodelink-controller"} {
		env, found := all[name]
		if !found {
			return nil, fmt.Errorf("deployment %s/%s is missing required MAPI container %s", deploy.Namespace, deploy.Name, name)
		}

		selected[name] = env
	}

	if env, found := all["machine-healthcheck-controller"]; found {
		selected["machine-healthcheck-controller"] = env
	}

	return selected, nil
}

// WaitForConfiguredProxyStatus waits for configured proxy endpoints to propagate.
func WaitForConfiguredProxyStatus(ctx context.Context, c client.Client) configv1.ProxyStatus {
	var (
		status configv1.ProxyStatus
		err    error
	)

	Eventually(func() error {
		status, err = GetConfiguredProxyStatus(ctx, c)
		return err
	}, WaitOverMedium).Should(Succeed(), "Cluster proxy endpoints did not propagate to status")

	return status
}

func proxyEnvVarsFromStatus(status configv1.ProxyStatus) []corev1.EnvVar {
	var env []corev1.EnvVar

	for _, variable := range []corev1.EnvVar{
		{Name: "HTTP_PROXY", Value: status.HTTPProxy},
		{Name: "HTTPS_PROXY", Value: status.HTTPSProxy},
		{Name: "NO_PROXY", Value: status.NoProxy},
	} {
		if variable.Value != "" {
			env = append(env, variable)
		}
	}

	return env
}

// WaitForAllContainersNoProxyEnvVars waits for all containers in the given Deployment
// to have HTTP_PROXY, HTTPS_PROXY, and NO_PROXY environment variables removed.
func WaitForAllContainersNoProxyEnvVars(c client.Client, namespace, deploymentName string) {
	kom := komega.New(c)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
	}

	By(fmt.Sprintf("Waiting for all containers in %s/%s to reflect unconfigured cluster-wide proxy", namespace, deploymentName))

	Eventually(kom.Object(deploy), WaitOverMedium).Should(
		WithTransform(proxyEnvVarsByContainer, HaveEach(BeEmpty())),
		fmt.Sprintf("Cluster-wide proxy environment variables were still set on some containers of %s/%s.", namespace, deploymentName),
	)
}

// proxyEnvVarsByContainer extracts proxy variables while retaining entries for containers without any.
func proxyEnvVarsByContainer(deploy *appsv1.Deployment) map[string][]corev1.EnvVar {
	result := make(map[string][]corev1.EnvVar, len(deploy.Spec.Template.Spec.Containers))

	for _, container := range deploy.Spec.Template.Spec.Containers {
		result[container.Name] = nil

		for _, env := range container.Env {
			switch env.Name {
			case "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY":
				result[container.Name] = append(result[container.Name], env)
			}
		}
	}

	return result
}

// DeleteProxy delete the MITM Proxy from the cluster.
func DeleteProxy(c client.Client, gomegaArgs ...interface{}) {
	ctx := context.Background()
	kom := komega.New(c)

	mitmSignerSecret := corev1resourcebuilder.Secret().WithName(mitmSignerName).WithNamespace(proxyNamespace).Build()
	mitmBootstrapConfigMap := corev1resourcebuilder.ConfigMap().WithName(mitmBootstrapName).WithNamespace(proxyNamespace).Build()
	mitmCustomPkiConfigMap := corev1resourcebuilder.ConfigMap().WithName(mitmCustomPKIName).WithNamespace(mitmCustomPKINamespace).Build()
	mitmDaemonset := appsv1resourcebuilder.DaemonSet().WithName(mitmDaemonsetName).WithNamespace(proxyNamespace).Build()
	mitmService := corev1resourcebuilder.Service().WithName(mitmServiceName).WithNamespace(proxyNamespace).Build()

	By("Deleting the MITM proxy Secret")
	Eventually(func() error { return client.IgnoreNotFound(c.Delete(ctx, mitmSignerSecret)) }).Should(Succeed(), "timed out deleting the MITM proxy Secret.")

	By("Deleting the MITM proxy ConfigMaps")
	Eventually(func() error { return client.IgnoreNotFound(c.Delete(ctx, mitmBootstrapConfigMap)) }).Should(Succeed(), "timed out deleting the MITM proxy Bootstrap ConfigMap.")
	Eventually(func() error { return client.IgnoreNotFound(c.Delete(ctx, mitmCustomPkiConfigMap)) }).Should(Succeed(), "timed out deleting the MITM proxy Custom PKI ConfigMap.")

	By("Deleting the MITM proxy DaemonSet")
	Eventually(func() error { return client.IgnoreNotFound(c.Delete(ctx, mitmDaemonset)) }).Should(Succeed(), "timed out deleting the MITM proxy DaemonSet.")

	By("Deleting the MITM proxy Service")
	Eventually(func() error { return client.IgnoreNotFound(c.Delete(ctx, mitmService)) }).Should(Succeed(), "timed out deleting the MITM proxy Service.")

	By("Checking that the MITM proxy components are removed from the cluster")

	Eventually(kom.Get(mitmSignerSecret)).
		Should(MatchError(ContainSubstring("not found")), "expected MITM proxy Secret to be removed from the cluster")

	Eventually(kom.Get(mitmBootstrapConfigMap)).
		Should(MatchError(ContainSubstring("not found")), "expected MITM proxy Bootstrap ConfigMap to be removed from the cluster")

	Eventually(kom.Get(mitmCustomPkiConfigMap)).
		Should(MatchError(ContainSubstring("not found")), "expected MITM proxy PKI ConfigMap to be removed from the cluster")

	Eventually(kom.Get(mitmDaemonset)).
		Should(MatchError(ContainSubstring("not found")), "expected MITM proxy DaemonSet to be removed from the cluster")

	Eventually(kom.Get(mitmService)).
		Should(MatchError(ContainSubstring("not found")), "expected MITM proxy Service to be removed from the cluster")
}

func buildDaemonSetVolumes() []corev1.Volume {
	mitmBootstrapPerms := int32(511)

	return []corev1.Volume{
		{
			Name: "mitm-bootstrap",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mitmBootstrapName,
					},
					DefaultMode: &mitmBootstrapPerms,
				},
			},
		},
		{
			Name: mitmSignerName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: mitmSignerName,
				},
			},
		},
		{
			Name: "mitm-workdir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func buildDaemonSetContainers() []corev1.Container {
	return []corev1.Container{{
		Name:            "proxy",
		Image:           "registry.redhat.io/ubi8/ubi",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c", "/root/startup.sh"},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 80,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      mitmBootstrapName,
				ReadOnly:  false,
				MountPath: "/root/startup.sh",
				SubPath:   "startup.sh",
			},
			{
				Name:      mitmSignerName,
				ReadOnly:  false,
				MountPath: "/root/certs",
			},
			{
				Name:      "mitm-workdir",
				ReadOnly:  false,
				MountPath: "/.mitmproxy",
			},
		},
	}}
}

func buildServicePorts() []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Protocol: "TCP",
			Port:     8080,
			TargetPort: intstr.IntOrString{
				IntVal: 8080,
			},
		},
	}
}

// generateCert generates an RSA private key and its corresponding X.509 certificate.
// https://golang.org/src/crypto/tls/generate_cert.go as a function
func generateCert() ([]byte, []byte, error) {
	var err error

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate rsa key: %w", err)
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, keyBytes, fmt.Errorf("failed to marshal private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, keyBytes, fmt.Errorf("failed to generate random serial number: %w", err)
	}

	keyID, err := func() ([]byte, error) {
		bytes := make([]byte, 20)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}

		return bytes, nil
	}()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate x509 certificate key ID: %w", err)
	}

	notBefore := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "mitm-proxy-ca",
		},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(time.Hour),
		SubjectKeyId:          keyID,
		AuthorityKeyId:        keyID,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate X.509 certificate: %w", err)
	}

	return certBytes, keyBytes, nil
}

// WaitForCAPIProxyEnvVars waits for the capi-controllers container to match Proxy status.
func WaitForCAPIProxyEnvVars(ctx context.Context, c client.Client, status configv1.ProxyStatus) {
	Eventually(func() (map[string][]corev1.EnvVar, error) {
		return capiProxyEnvVars(ctx, c)
	}, WaitOverMedium).Should(
		HaveEach(ConsistOf(proxyEnvVarsFromStatus(status))),
		"The capi-controllers container did not reflect Proxy status",
	)
}

func capiProxyEnvVars(ctx context.Context, c client.Client) (map[string][]corev1.EnvVar, error) {
	const controllerName = "capi-controllers"

	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{Namespace: ClusterAPINamespace, Name: controllerName}

	if err := c.Get(ctx, key, deployment); err != nil {
		return nil, fmt.Errorf("getting CAPI Deployment %s: %w", key, err)
	}

	env, found := proxyEnvVarsByContainer(deployment)[controllerName]
	if !found {
		return nil, fmt.Errorf("deployment %s is missing required container %s", key, controllerName)
	}

	return map[string][]corev1.EnvVar{key.String() + "/" + controllerName: env}, nil
}
