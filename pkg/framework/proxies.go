package framework

import (
	"context"
	"errors"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const proxySetup = `
cd /.mitmproxy
cat /root/certs/tls.key /root/certs/tls.crt > /.mitmproxy/mitmproxy-ca.pem
curl -O https://snapshots.mitmproxy.org/5.3.0/mitmproxy-5.3.0-linux.tar.gz
tar xvf mitmproxy-5.3.0-linux.tar.gz
HOME=/.mitmproxy ./mitmdump
`

const mitmSignerCert = `
-----BEGIN CERTIFICATE-----
MIICQDCCAamgAwIBAgIUXyjCAhFKon7pffzuFrHRBmLQRqwwDQYJKoZIhvcNAQEL
BQAwMjELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAk5DMRYwFAYDVQQDDA1taXRtLXBy
b3h5LWNhMB4XDTIwMTEyMDIxMTk1NVoXDTQ4MDUwMTIxMTk1NVowMjELMAkGA1UE
BhMCVVMxCzAJBgNVBAgMAk5DMRYwFAYDVQQDDA1taXRtLXByb3h5LWNhMIGfMA0G
CSqGSIb3DQEBAQUAA4GNADCBiQKBgQDPqHuUZz1Qt236e03XWjsVCy1OnuuWQ3a1
OQX21waqypGPem6iKiSmAmf+YrKdkX2O5L2sqPjpZ+civ9z5h9d0xCCNy+06UQTu
+pIrgIxm2p5wQmSGgI5KNUL6dgU3dC5aRDMU7hV0RGXKrJktTmU2SnZsOUe4SZ5L
Vyanpk8N1QIDAQABo1MwUTAdBgNVHQ4EFgQUFD2D/LygThvwLRKukmH1Y++hxLIw
HwYDVR0jBBgwFoAUFD2D/LygThvwLRKukmH1Y++hxLIwDwYDVR0TAQH/BAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOBgQCLZeGgfFRCOWqSXcDzUzeJE03HD/y2rLY+hDMn
kj69xY3+K0z4DRbb1vpMiYsTYF/Z1d6blND4KW+7oi9R6PPYg6XATDMf9tVfTQIe
qBkDNfqABPlwJABpixwD20XXQUBqADEyO3tdQLtiMi5Qr6QHOX3+FepiHAgdxAFt
Mqy4Gw==
-----END CERTIFICATE-----
`
const mitmSignerKey = `
-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDPqHuUZz1Qt236e03XWjsVCy1OnuuWQ3a1OQX21waqypGPem6i
KiSmAmf+YrKdkX2O5L2sqPjpZ+civ9z5h9d0xCCNy+06UQTu+pIrgIxm2p5wQmSG
gI5KNUL6dgU3dC5aRDMU7hV0RGXKrJktTmU2SnZsOUe4SZ5LVyanpk8N1QIDAQAB
AoGBAJksuGuRc8MUawV26sZNgoNVSUhTJYgjn26x71pS5lIZNiHVt8HawEnMQJV+
jC56YVmEFP1FbsYMpIwXZpKRxzS/uXInKUbRlYQuJ8kVG7a0foe+vja1gxmG4b+Z
69V6S+4Y8AhHEcw+Ek04LUWOzEV0NpBa8VJYVHtW3bTarE0JAkEA9DBvOmkg9rit
x2M9zLR+BCPvgP16EARY2Ik+ZlOP7hC2iPL9mEjxSujJ78SSrrY09hPomemAqBz6
eiWbsL9KgwJBANmzt+pOGu0fON3MFbjRKfZHAdsDJQipTEGKr5jig8nZg84R+zBA
rfm9Vm2zdHaZ6rGTpMv62roHrXnq9y1htscCQQDQo8GlqsWbiNgSkNzw1xcE+p9d
Gzb8EHrJKRrD24oS4vzTrqq3PzvLwXMpBlA+Lzi5OPF48GYZPglV7GRGdGt5AkAY
7v53dW6cDeFjdcZfHoWh0UwjG18YeNtk/k9SQU86xRDVfzW3txC187t8YPtLwiEh
KXnMavS2Lb7uobyhk/ltAkEA48NjOS7IL1JMR2X/r5uBrEQhd/XjnaYePdav32Ii
2ygHNMMkN1/ueVBCypFaU2UZDVNabyZ+MJYnR8dK4xxpQg==
-----END RSA PRIVATE KEY-----
`

// DeployClusterProxy Deploys an HTTP proxy to the proxy node
func DeployClusterProxy(c runtimeclient.Client) error {
	mitmDeploymentLabels := map[string]string{
		"app": "mitm-proxy",
	}

	objectMeta := metav1.ObjectMeta{
		Name:      "mitm-proxy",
		Namespace: MachineAPINamespace,
		Labels:    mitmDeploymentLabels,
	}

	mitmSigner := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-signer",
			Namespace: MachineAPINamespace,
			Labels:    mitmDeploymentLabels,
		},
		Data: map[string][]byte{
			"tls.crt": []byte(mitmSignerCert),
			"tls.key": []byte(mitmSignerKey),
		},
	}

	err := c.Create(context.Background(), &mitmSigner)

	var mitmBootstrapPerms int32 = 511
	mitmBootstrapConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-bootstrap",
			Namespace: MachineAPINamespace,
		},
		Data: map[string]string{
			"startup.sh": proxySetup,
		},
	}

	err = c.Create(context.Background(), &mitmBootstrapConfigMap)
	if err != nil {
		return err
	}

	mitmCustomPkiConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-custom-pki",
			Namespace: "openshift-config",
		},
		Data: map[string]string{
			"ca-bundle.crt": mitmSignerCert,
		},
	}

	err = c.Create(context.Background(), &mitmCustomPkiConfigMap)
	if err != nil {
		return err
	}

	daemonset := &appsv1.DaemonSet{
		ObjectMeta: objectMeta,
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: mitmDeploymentLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objectMeta,
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "mitm-bootstrap",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "mitm-bootstrap",
									},
									DefaultMode: &mitmBootstrapPerms,
								},
							},
						},
						{
							Name: "mitm-signer",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "mitm-signer",
								},
							},
						},
						{
							Name: "mitm-workdir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "proxy",
							Image:   "registry.redhat.io/ubi8/ubi",
							Command: []string{"/bin/sh", "-c", "/root/startup.sh"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "mitm-bootstrap",
									ReadOnly:  false,
									MountPath: "/root/startup.sh",
									SubPath:   "startup.sh",
								},
								{
									Name:      "mitm-signer",
									ReadOnly:  false,
									MountPath: "/root/certs",
								},
								{
									Name:      "mitm-workdir",
									ReadOnly:  false,
									MountPath: "/.mitmproxy",
								},
							},
						},
					},
				},
			},
		},
	}
	err = c.Create(context.Background(), daemonset)
	if err != nil {
		return err
	}

	if !IsDaemonsetAvailable(c, objectMeta.Name, objectMeta.Namespace) {
		return errors.New("daemonset did not become available")
	}

	service := &corev1.Service{
		ObjectMeta: objectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     8080,
					TargetPort: intstr.IntOrString{
						IntVal: 8080,
					},
				},
			},
			Selector: mitmDeploymentLabels,
		},
	}
	err = c.Create(context.Background(), service)
	if err != nil {
		return err
	}
	if !IsServiceAvailable(c, objectMeta.Name, objectMeta.Namespace) {
		return errors.New("service did not become available")
	}

	return err
}

// DestroyClusterProxy destroys the HTTP proxy and associated resources
func DestroyClusterProxy(c runtimeclient.Client) error {
	mitmDeploymentLabels := map[string]string{
		"app": "mitm-proxy",
	}

	mitmObjectMeta := metav1.ObjectMeta{
		Name:      "mitm-proxy",
		Namespace: MachineAPINamespace,
		Labels:    mitmDeploymentLabels,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-bootstrap",
			Namespace: MachineAPINamespace,
		},
	}
	if err := c.Delete(context.Background(), configMap); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-custom-pki",
			Namespace: "openshift-config",
		},
	}
	if err := c.Delete(context.Background(), configMap); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mitm-signer",
			Namespace: MachineAPINamespace,
		},
	}
	if err := c.Delete(context.Background(), secret); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	daemonset := &appsv1.DaemonSet{
		ObjectMeta: mitmObjectMeta,
	}
	if err := c.Delete(context.Background(), daemonset); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: mitmObjectMeta,
	}
	if err := c.Delete(context.Background(), service); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// WaitForProxyInjectionSync waits for the deployment to sync with the state of the cluster-proxy
func WaitForProxyInjectionSync(c runtimeclient.Client, name, namespace string, shouldBePresent bool) (bool, error) {
	if err := wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		deployment, err := GetDeployment(c, name, namespace)
		if err != nil {
			return false, nil
		}
		hasHTTPProxy := false
		hasHTTPSProxy := false
		hasNoProxy := false
		for _, container := range deployment.Spec.Template.Spec.Containers {
			for _, envVar := range container.Env {
				switch envVar.Name {
				case "NO_PROXY":
					hasNoProxy = true
				case "HTTPS_PROXY":
					hasHTTPSProxy = true
				case "HTTP_PROXY":
					hasHTTPProxy = true
				}
			}
		}
		return (hasHTTPProxy &&
			hasHTTPSProxy &&
			hasNoProxy) == shouldBePresent, nil
	}); err != nil {
		return false, fmt.Errorf("error checking isDeploymentAvailable: %v", err)
	}
	return true, nil
}

// GetClusterProxy fetches the global cluster proxy object.
func GetClusterProxy(c runtimeclient.Client) (*configv1.Proxy, error) {
	proxy := &configv1.Proxy{}
	proxyName := runtimeclient.ObjectKey{
		Name: GlobalInfrastuctureName,
	}

	if err := c.Get(context.Background(), proxyName, proxy); err != nil {
		return nil, err
	}

	return proxy, nil
}
