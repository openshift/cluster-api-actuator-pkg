/*
Copyright 2026 Red Hat, Inc.

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

package util

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const proxyResourceName = "cluster"

// GetProxyEnvVars reads the cluster-wide Proxy singleton and returns the
// standard proxy environment variables as a slice. Returns an empty slice
// when no proxy is configured (Proxy CR missing or all fields empty).
func GetProxyEnvVars(ctx context.Context, cl client.Reader) ([]corev1.EnvVar, error) {
	proxy := &configv1.Proxy{}
	if err := cl.Get(ctx, client.ObjectKey{Name: proxyResourceName}, proxy); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get cluster proxy: %w", err)
	}

	var vars []corev1.EnvVar

	if proxy.Status.HTTPProxy != "" {
		vars = append(vars, corev1.EnvVar{Name: "HTTP_PROXY", Value: proxy.Status.HTTPProxy})
	}

	if proxy.Status.HTTPSProxy != "" {
		vars = append(vars, corev1.EnvVar{Name: "HTTPS_PROXY", Value: proxy.Status.HTTPSProxy})
	}

	if proxy.Status.NoProxy != "" {
		vars = append(vars, corev1.EnvVar{Name: "NO_PROXY", Value: proxy.Status.NoProxy})
	}

	return vars, nil
}
