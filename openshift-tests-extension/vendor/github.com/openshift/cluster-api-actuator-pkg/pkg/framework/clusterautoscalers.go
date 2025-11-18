package framework

import (
	"context"
	"fmt"

	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClusterAutoscaler gets a ClusterAutoscaler by its name from the default machine API namespace.
func GetClusterAutoscaler(client runtimeclient.Client, name string) (*caov1.ClusterAutoscaler, error) {
	clusterAutoscaler := &caov1.ClusterAutoscaler{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: name}

	if err := client.Get(context.Background(), key, clusterAutoscaler); err != nil {
		return nil, fmt.Errorf("error querying api for ClusterAutoscaler object: %w", err)
	}

	return clusterAutoscaler, nil
}
