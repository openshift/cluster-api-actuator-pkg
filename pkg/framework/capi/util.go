package capi

import (
	"fmt"
	"net/url"
	"strconv"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetControlPlaneHostAndPort(cl client.Client) (string, int32, error) {
	var infraCluster configv1.Infrastructure

	namespacedName := client.ObjectKey{
		Namespace: CAPINamespace,
		Name:      "cluster",
	}

	if err := cl.Get(ctx, namespacedName, &infraCluster); err != nil {
		return "", 0, fmt.Errorf("failed to get the infrastructure object: %w", err)
	}

	apiURL, err := url.Parse(infraCluster.Status.APIServerURL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse the API server URL: %w", err)
	}

	port, err := strconv.ParseInt(apiURL.Port(), 10, 32)
	if err != nil {
		return apiURL.Hostname(), 0, fmt.Errorf("failed to parse port: %w", err)
	}

	return apiURL.Hostname(), int32(port), nil
}
