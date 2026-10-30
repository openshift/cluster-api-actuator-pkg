package framework

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	httpProxyEnvName     = "HTTP_PROXY"
	httpsProxyEnvName    = "HTTPS_PROXY"
	noProxyBypass        = ".internal"
	proxyResourceName    = "proxy"
	testControllerName   = "controller"
	unrelatedEnvName     = "UNRELATED"
	wrongProxyValue      = "http://wrong"
	managerContainerName = "manager"
	noProxyEnvName       = "NO_PROXY"
	testProxyURL         = "http://proxy"
	testSecureProxyURL   = "http://secure-proxy"
	sidecarContainerName = "sidecar"
	testEnvValue         = "value"
)

func TestProxyEnvVarsByContainer(t *testing.T) {
	proxyEnv := []corev1.EnvVar{
		{Name: httpProxyEnvName, Value: testProxyURL},
		{Name: httpsProxyEnvName, Value: testSecureProxyURL},
		{Name: noProxyEnvName, Value: noProxyBypass},
		{Name: httpProxyEnvName, Value: "http://duplicate"},
		{Name: httpsProxyEnvName, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: proxyResourceName},
				Key:                  "url",
			},
		}},
		{Name: noProxyEnvName},
	}

	tests := []struct {
		name       string
		containers []corev1.Container
		want       map[string][]corev1.EnvVar
	}{
		{name: "no containers", want: map[string][]corev1.EnvVar{}},
		{
			name:       "containers without environment variables retained",
			containers: []corev1.Container{{Name: testControllerName}, {Name: sidecarContainerName}},
			want:       map[string][]corev1.EnvVar{testControllerName: nil, sidecarContainerName: nil},
		},
		{
			name: "unrelated variables ignored",
			containers: []corev1.Container{{Name: testControllerName, Env: []corev1.EnvVar{
				{Name: unrelatedEnvName, Value: testEnvValue},
				{Name: "HTTP_PROXY_OTHER", Value: testEnvValue},
			}}},
			want: map[string][]corev1.EnvVar{testControllerName: nil},
		},
		{
			name: "proxy entries preserved and grouped by container",
			containers: []corev1.Container{
				{Name: testControllerName, Env: append([]corev1.EnvVar{{Name: unrelatedEnvName, Value: testEnvValue}}, proxyEnv...)},
				{Name: "other-controller", Env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: "http://other"}}},
				{Name: sidecarContainerName},
			},
			want: map[string][]corev1.EnvVar{
				testControllerName:   proxyEnv,
				"other-controller":   {{Name: httpProxyEnvName, Value: "http://other"}},
				sidecarContainerName: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: tt.containers},
				}},
			}
			before := deployment.DeepCopy()

			got := proxyEnvVarsByContainer(deployment)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("proxy variables = %#v, want %#v", got, tt.want)
			}

			if !reflect.DeepEqual(deployment, before) {
				t.Fatal("extracting proxy variables modified the deployment")
			}
		})
	}
}

func TestClusterProxyConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		proxy       *configv1.Proxy
		wantEnabled bool
		wantReady   bool
		wantError   bool
	}{
		{name: "missing proxy", wantError: true},
		{name: "unconfigured", proxy: &configv1.Proxy{}},
		{name: "bypass only", proxy: &configv1.Proxy{Status: configv1.ProxyStatus{NoProxy: noProxyBypass}}},
		{
			name: "HTTP spec only", wantEnabled: true,
			proxy: &configv1.Proxy{Spec: configv1.ProxySpec{HTTPProxy: testProxyURL}},
		},
		{
			name: "HTTPS spec only", wantEnabled: true,
			proxy: &configv1.Proxy{Spec: configv1.ProxySpec{HTTPSProxy: testProxyURL}},
		},
		{
			name: "HTTP status only", wantEnabled: true,
			proxy: &configv1.Proxy{Status: configv1.ProxyStatus{HTTPProxy: testProxyURL}},
		},
		{
			name: "HTTPS status only", wantEnabled: true,
			proxy: &configv1.Proxy{Status: configv1.ProxyStatus{HTTPSProxy: testProxyURL}},
		},
		{
			name: "HTTP ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: testProxyURL, NoProxy: noProxyBypass},
				Status: configv1.ProxyStatus{HTTPProxy: testProxyURL, NoProxy: ".internal,.svc"},
			},
		},
		{
			name: "HTTPS ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPSProxy: testProxyURL},
				Status: configv1.ProxyStatus{HTTPSProxy: testProxyURL},
			},
		},
		{
			name: "both endpoints ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: testProxyURL, HTTPSProxy: testSecureProxyURL},
				Status: configv1.ProxyStatus{HTTPProxy: testProxyURL, HTTPSProxy: testSecureProxyURL},
			},
		},
		{
			name: "stale HTTP status", wantEnabled: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: "http://new"},
				Status: configv1.ProxyStatus{HTTPProxy: "http://old"},
			},
		},
		{
			name: "stale HTTPS status", wantEnabled: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPSProxy: "http://new"},
				Status: configv1.ProxyStatus{HTTPSProxy: "http://old"},
			},
		},
		{
			name: "only one endpoint propagated", wantEnabled: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: testProxyURL, HTTPSProxy: testProxyURL},
				Status: configv1.ProxyStatus{HTTPProxy: testProxyURL},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &proxyTestClient{proxy: tt.proxy}

			enabled, err := IsClusterProxyEnabled(context.Background(), c)
			if enabled != tt.wantEnabled || (err != nil) != tt.wantError {
				t.Fatalf("enabled = %v, error = %v; want enabled = %v, error = %v", enabled, err, tt.wantEnabled, tt.wantError)
			}

			status, err := GetConfiguredProxyStatus(context.Background(), c)
			if (err == nil) != tt.wantReady {
				t.Fatalf("status = %+v, error = %v; want ready = %v", status, err, tt.wantReady)
			}

			if tt.wantReady && (status.HTTPProxy != tt.proxy.Status.HTTPProxy || status.HTTPSProxy != tt.proxy.Status.HTTPSProxy || status.NoProxy != tt.proxy.Status.NoProxy) {
				t.Fatalf("returned status %+v does not match cluster Proxy status %+v", status, tt.proxy.Status)
			}
		})
	}
}

func TestContainerProxyEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		status    configv1.ProxyStatus
		env       []corev1.EnvVar
		wantError bool
	}{
		{name: "no proxy"},
		{
			name:   "all proxy values",
			status: configv1.ProxyStatus{HTTPProxy: testProxyURL, HTTPSProxy: testSecureProxyURL, NoProxy: noProxyBypass},
			env: []corev1.EnvVar{
				{Name: httpProxyEnvName, Value: testProxyURL},
				{Name: httpsProxyEnvName, Value: testSecureProxyURL},
				{Name: noProxyEnvName, Value: noProxyBypass},
				{Name: unrelatedEnvName, Value: "preserved"},
			},
		},
		{name: "HTTP only", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}, env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: testProxyURL}}},
		{name: "HTTPS only", status: configv1.ProxyStatus{HTTPSProxy: testProxyURL}, env: []corev1.EnvVar{{Name: httpsProxyEnvName, Value: testProxyURL}}},
		{name: "bypass only", status: configv1.ProxyStatus{NoProxy: noProxyBypass}, env: []corev1.EnvVar{{Name: noProxyEnvName, Value: noProxyBypass}}},
		{name: "wrong value", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}, env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: wrongProxyValue}}, wantError: true},
		{name: "missing value", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}, wantError: true},
		{name: "stale value", env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: testProxyURL}}, wantError: true},
		{name: "empty variable must be absent", env: []corev1.EnvVar{{Name: httpsProxyEnvName}}, wantError: true},
		{name: "duplicate variable", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}, env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: testProxyURL}, {Name: httpProxyEnvName, Value: testProxyURL}}, wantError: true},
		{name: "value reference", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}, env: []corev1.EnvVar{{Name: httpProxyEnvName, ValueFrom: &corev1.EnvVarSource{}}}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: managerContainerName, Env: tt.env}},
				}},
			}}
			actual := proxyEnvVarsByContainer(deployment)[managerContainerName]

			matched, err := gomega.ConsistOf(proxyEnvVarsFromStatus(tt.status)).Match(actual)
			if err != nil || matched == tt.wantError {
				t.Fatalf("matched = %v, error = %v; want error = %v", matched, err, tt.wantError)
			}
		})
	}
}

func TestProxyEnvVarsMismatchDiagnostics(t *testing.T) {
	actual := map[string][]corev1.EnvVar{
		managerContainerName: {{Name: httpProxyEnvName, Value: wrongProxyValue}},
	}
	expected := proxyEnvVarsFromStatus(configv1.ProxyStatus{HTTPProxy: "http://expected"})
	matcher := gomega.HaveEach(gomega.ConsistOf(expected))

	matched, err := matcher.Match(actual)
	if err != nil || matched {
		t.Fatalf("expected a mismatch, got matched = %v, error = %v", matched, err)
	}

	message := matcher.FailureMessage(actual)

	for _, detail := range []string{managerContainerName, httpProxyEnvName, wrongProxyValue, "http://expected"} {
		if !strings.Contains(message, detail) {
			t.Errorf("failure message %q does not contain %q", message, detail)
		}
	}
}

func TestMAPIControllersProxyEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		status    configv1.ProxyStatus
		mutate    func(*appsv1.Deployment)
		wantError bool
	}{
		{name: "controllers configured and metrics sidecars excluded"},
		{name: "HTTP only", status: configv1.ProxyStatus{HTTPProxy: testProxyURL}},
		{name: "HTTPS only", status: configv1.ProxyStatus{HTTPSProxy: testProxyURL}},
		{name: "optional healthcheck absent", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers[:3], d.Spec.Template.Spec.Containers[4:]...)
		}},
		{name: "empty deployment", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = nil
		}, wantError: true},
		{name: "sidecars only", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = d.Spec.Template.Spec.Containers[4:]
		}, wantError: true},
		{name: "missing machine controller", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = d.Spec.Template.Spec.Containers[1:]
		}, wantError: true},
		{name: "missing machineset controller", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers[:1], d.Spec.Template.Spec.Containers[2:]...)
		}, wantError: true},
		{name: "missing nodelink controller", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers[:2], d.Spec.Template.Spec.Containers[3:]...)
		}, wantError: true},
		{name: "machine controller missing proxy variables", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Env = nil
		}, wantError: true},
		{name: "machineset controller missing proxy variables", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[1].Env = nil
		}, wantError: true},
		{name: "nodelink controller missing proxy variables", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[2].Env = nil
		}, wantError: true},
		{name: "healthcheck controller missing proxy variables", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[3].Env = nil
		}, wantError: true},
		{name: "wrong proxy value", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Env[0].Value = wrongProxyValue
		}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.status
			if status.HTTPProxy == "" && status.HTTPSProxy == "" {
				status = configv1.ProxyStatus{HTTPProxy: testProxyURL, HTTPSProxy: testSecureProxyURL, NoProxy: noProxyBypass}
			}

			deployment := &appsv1.Deployment{}

			for _, name := range []string{"machine-controller", "machineset-controller", "nodelink-controller", "machine-healthcheck-controller"} {
				container := corev1.Container{Name: name}

				for key, value := range map[string]string{httpProxyEnvName: status.HTTPProxy, httpsProxyEnvName: status.HTTPSProxy, noProxyEnvName: status.NoProxy} {
					if value != "" {
						container.Env = append(container.Env, corev1.EnvVar{Name: key, Value: value})
					}
				}

				deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
			}

			for _, name := range []string{"kube-rbac-proxy-machineset-mtrc", "kube-rbac-proxy-machine-mtrc", "kube-rbac-proxy-mhc-mtrc"} {
				deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, corev1.Container{Name: name})
			}

			if tt.mutate != nil {
				tt.mutate(deployment)
			}

			actual, err := mapiControllersProxyEnvVars(deployment)

			matched := false

			if err == nil {
				matched, err = gomega.HaveEach(gomega.ConsistOf(proxyEnvVarsFromStatus(status))).Match(actual)
			}

			if (err != nil || !matched) != tt.wantError {
				t.Fatalf("matched = %v, error = %v; want error = %v", matched, err, tt.wantError)
			}
		})
	}
}

func TestCAPIProxyEnvVars(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "capi-controllers", Namespace: ClusterAPINamespace},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{
				{Name: "capi-controllers", Env: []corev1.EnvVar{{Name: httpProxyEnvName, Value: testProxyURL}}},
				{Name: sidecarContainerName},
			}},
		}},
	}

	tests := []struct {
		name      string
		mutate    func(*appsv1.Deployment)
		wantError bool
	}{
		{name: "no labels or annotations required", mutate: func(_ *appsv1.Deployment) {}},
		{name: "annotation cannot redirect target", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Annotations = map[string]string{"capi-operator.openshift.io/inject-proxy": sidecarContainerName}
		}},
		{name: "other deployment excluded", mutate: func(d *appsv1.Deployment) { d.Name = "provider" }, wantError: true},
		{name: "other namespace excluded", mutate: func(d *appsv1.Deployment) { d.Namespace = "other" }, wantError: true},
		{name: "required container missing", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Name = managerContainerName
		}, wantError: true},
		{name: "missing proxy without annotation still fails", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Env = nil
		}, wantError: true},
		{name: "stale proxy config", mutate: func(d *appsv1.Deployment) {
			d.Spec.Template.Spec.Containers[0].Env[0].Value = "http://stale"
		}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := deployment.DeepCopy()
			tt.mutate(d)

			c := &proxyTestClient{deployments: []appsv1.Deployment{*d}}

			actual, err := capiProxyEnvVars(context.Background(), c)

			matched := false

			if err == nil {
				expected := proxyEnvVarsFromStatus(configv1.ProxyStatus{HTTPProxy: testProxyURL})
				matched, err = gomega.HaveEach(gomega.ConsistOf(expected)).Match(actual)
			}

			if (err != nil || !matched) != tt.wantError {
				t.Fatalf("matched = %v, error = %v; want error = %v", matched, err, tt.wantError)
			}
		})
	}
}

type proxyTestClient struct {
	client.Client
	proxy       *configv1.Proxy
	version     *configv1.ClusterVersion
	gate        *configv1.FeatureGate
	clusterAPI  *operatorv1alpha1.ClusterAPI
	deployments []appsv1.Deployment
}

func (c *proxyTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch out := obj.(type) {
	case *appsv1.Deployment:
		for _, deployment := range c.deployments {
			if client.ObjectKeyFromObject(&deployment) == key {
				*out = *deployment.DeepCopy()

				return nil
			}
		}

	case *configv1.Proxy:
		if c.proxy != nil {
			*out = *c.proxy.DeepCopy()

			return nil
		}
	case *configv1.ClusterVersion:
		if c.version != nil {
			*out = *c.version.DeepCopy()

			return nil
		}
	case *configv1.FeatureGate:
		if c.gate != nil {
			*out = *c.gate.DeepCopy()

			return nil
		}
	case *operatorv1alpha1.ClusterAPI:
		if c.clusterAPI != nil {
			*out = *c.clusterAPI.DeepCopy()

			return nil
		}
	}

	return fmt.Errorf("test object %T unavailable", obj)
}

func (c *proxyTestClient) List(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
	options := (&client.ListOptions{}).ApplyOptions(opts)

	deployments, ok := list.(*appsv1.DeploymentList)
	if !ok {
		return fmt.Errorf("unexpected list type %T", list)
	}

	deployments.Items = nil

	for _, deployment := range c.deployments {
		if options.Namespace != "" && options.Namespace != deployment.Namespace {
			continue
		}

		if options.LabelSelector != nil && !options.LabelSelector.Matches(labels.Set(deployment.Labels)) {
			continue
		}

		deployments.Items = append(deployments.Items, *deployment.DeepCopy())
	}

	return nil
}
