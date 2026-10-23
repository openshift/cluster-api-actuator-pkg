package framework

import (
	"context"
	"fmt"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		{name: "bypass only", proxy: &configv1.Proxy{Status: configv1.ProxyStatus{NoProxy: ".internal"}}},
		{
			name: "HTTP spec only", wantEnabled: true,
			proxy: &configv1.Proxy{Spec: configv1.ProxySpec{HTTPProxy: "http://proxy"}},
		},
		{
			name: "HTTPS spec only", wantEnabled: true,
			proxy: &configv1.Proxy{Spec: configv1.ProxySpec{HTTPSProxy: "http://proxy"}},
		},
		{
			name: "HTTP status only", wantEnabled: true,
			proxy: &configv1.Proxy{Status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}},
		},
		{
			name: "HTTPS status only", wantEnabled: true,
			proxy: &configv1.Proxy{Status: configv1.ProxyStatus{HTTPSProxy: "http://proxy"}},
		},
		{
			name: "HTTP ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: "http://proxy", NoProxy: ".internal"},
				Status: configv1.ProxyStatus{HTTPProxy: "http://proxy", NoProxy: ".internal,.svc"},
			},
		},
		{
			name: "HTTPS ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPSProxy: "http://proxy"},
				Status: configv1.ProxyStatus{HTTPSProxy: "http://proxy"},
			},
		},
		{
			name: "both endpoints ready", wantEnabled: true, wantReady: true,
			proxy: &configv1.Proxy{
				Spec:   configv1.ProxySpec{HTTPProxy: "http://proxy", HTTPSProxy: "http://secure-proxy"},
				Status: configv1.ProxyStatus{HTTPProxy: "http://proxy", HTTPSProxy: "http://secure-proxy"},
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
				Spec:   configv1.ProxySpec{HTTPProxy: "http://proxy", HTTPSProxy: "http://proxy"},
				Status: configv1.ProxyStatus{HTTPProxy: "http://proxy"},
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
			status: configv1.ProxyStatus{HTTPProxy: "http://proxy", HTTPSProxy: "http://secure-proxy", NoProxy: ".internal"},
			env: []corev1.EnvVar{
				{Name: "HTTP_PROXY", Value: "http://proxy"},
				{Name: "HTTPS_PROXY", Value: "http://secure-proxy"},
				{Name: "NO_PROXY", Value: ".internal"},
				{Name: "UNRELATED", Value: "preserved"},
			},
		},
		{name: "HTTP only", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}, env: []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy"}}},
		{name: "HTTPS only", status: configv1.ProxyStatus{HTTPSProxy: "http://proxy"}, env: []corev1.EnvVar{{Name: "HTTPS_PROXY", Value: "http://proxy"}}},
		{name: "bypass only", status: configv1.ProxyStatus{NoProxy: ".internal"}, env: []corev1.EnvVar{{Name: "NO_PROXY", Value: ".internal"}}},
		{name: "wrong value", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}, env: []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://wrong"}}, wantError: true},
		{name: "missing value", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}, wantError: true},
		{name: "stale value", env: []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy"}}, wantError: true},
		{name: "empty variable must be absent", env: []corev1.EnvVar{{Name: "HTTPS_PROXY"}}, wantError: true},
		{name: "duplicate variable", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}, env: []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy"}, {Name: "HTTP_PROXY", Value: "http://proxy"}}, wantError: true},
		{name: "value reference", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}, env: []corev1.EnvVar{{Name: "HTTP_PROXY", ValueFrom: &corev1.EnvVarSource{}}}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkContainerProxyEnvVars(corev1.Container{Name: "manager", Env: tt.env}, tt.status)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
		})
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
		{name: "HTTP only", status: configv1.ProxyStatus{HTTPProxy: "http://proxy"}},
		{name: "HTTPS only", status: configv1.ProxyStatus{HTTPSProxy: "http://proxy"}},
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
			d.Spec.Template.Spec.Containers[0].Env[0].Value = "http://wrong"
		}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.status
			if status.HTTPProxy == "" && status.HTTPSProxy == "" {
				status = configv1.ProxyStatus{HTTPProxy: "http://proxy", HTTPSProxy: "http://secure-proxy", NoProxy: ".internal"}
			}

			deployment := &appsv1.Deployment{}

			for _, name := range []string{"machine-controller", "machineset-controller", "nodelink-controller", "machine-healthcheck-controller"} {
				container := corev1.Container{Name: name}

				for key, value := range map[string]string{"HTTP_PROXY": status.HTTPProxy, "HTTPS_PROXY": status.HTTPSProxy, "NO_PROXY": status.NoProxy} {
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

			err := checkMAPIControllersProxyEnvVars(deployment, status)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

func TestCAPIProxyEnvVars(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "provider", Namespace: ClusterAPINamespace, Labels: map[string]string{capiManagedLabel: "provider"}},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{capiProxyAnnotation: " manager, "}},
			Spec: corev1.PodSpec{Containers: []corev1.Container{
				{Name: "manager", Env: []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy"}}},
				{Name: "sidecar"},
			}},
		}},
	}

	tests := []struct {
		name      string
		mutate    func(*appsv1.Deployment)
		wantError bool
	}{
		{name: "only annotated containers checked", mutate: func(_ *appsv1.Deployment) {}},
		{name: "legacy provider label excluded", mutate: func(d *appsv1.Deployment) { d.Labels = map[string]string{"cluster.x-k8s.io/provider": "provider"} }, wantError: true},
		{name: "other namespace excluded", mutate: func(d *appsv1.Deployment) { d.Namespace = "other" }, wantError: true},
		{name: "no opt in", mutate: func(d *appsv1.Deployment) { d.Spec.Template.Annotations = nil }, wantError: true},
		{name: "empty opt in", mutate: func(d *appsv1.Deployment) { d.Spec.Template.Annotations[capiProxyAnnotation] = " , " }, wantError: true},
		{name: "unknown container", mutate: func(d *appsv1.Deployment) { d.Spec.Template.Annotations[capiProxyAnnotation] = "missing" }, wantError: true},
		{name: "all annotated containers checked", mutate: func(d *appsv1.Deployment) { d.Spec.Template.Annotations[capiProxyAnnotation] = "manager,sidecar" }, wantError: true},
		{name: "stale manager config", mutate: func(d *appsv1.Deployment) { d.Spec.Template.Spec.Containers[0].Env[0].Value = "http://stale" }, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := deployment.DeepCopy()
			tt.mutate(d)

			c := &proxyTestClient{deployments: []appsv1.Deployment{*d}}

			err := CheckCAPIProxyEnvVars(context.Background(), c, configv1.ProxyStatus{HTTPProxy: "http://proxy"})
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

type proxyTestClient struct {
	client.Client
	proxy       *configv1.Proxy
	version     *configv1.ClusterVersion
	gate        *configv1.FeatureGate
	clusterAPI  *unstructured.Unstructured
	deployments []appsv1.Deployment
}

func (c *proxyTestClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	switch out := obj.(type) {
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
	case *unstructured.Unstructured:
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
