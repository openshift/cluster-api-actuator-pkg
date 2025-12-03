package framework

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capiv1resourcebuilder "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder/cluster-api/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateCoreCluster creates a cluster with the given name and returns the cluster object.
func CreateCoreCluster(ctx context.Context, cl client.Client, clusterName, infraClusterKind string) *clusterv1beta1.Cluster {
	By("Creating core cluster")

	ref := &corev1.ObjectReference{
		APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		Kind:       infraClusterKind,
		Name:       clusterName,
		Namespace:  ClusterAPINamespace,
	}
	cluster := capiv1resourcebuilder.Cluster().WithName(clusterName).WithNamespace(ClusterAPINamespace).WithInfrastructureRef(ref).Build()

	if infraClusterKind == "VSphereCluster" {
		host, port, err := GetControlPlaneHostAndPort(ctx, cl)
		if err != nil {
			Expect(err).ToNot(HaveOccurred(), "Failed to get control plane host and port")
		}

		cluster.Spec.ControlPlaneEndpoint = clusterv1beta1.APIEndpoint{
			Host: host,
			Port: port,
		}
	}

	if err := cl.Create(ctx, cluster); err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred(), "Failed to create cluster")
	}

	Eventually(func() (bool, error) {
		patchedCluster := &clusterv1beta1.Cluster{}
		err := cl.Get(ctx, client.ObjectKeyFromObject(cluster), patchedCluster)
		if err != nil {
			return false, err
		}

		return v1beta1conditions.IsTrue(patchedCluster, clusterv1beta1.ControlPlaneInitializedCondition), nil
	}, WaitMedium).Should(BeTrue(), "it should be able to create cluster")

	return cluster
}
