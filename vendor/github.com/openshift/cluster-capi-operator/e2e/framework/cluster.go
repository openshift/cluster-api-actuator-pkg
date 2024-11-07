package framework

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateCluster creates a cluster with the given name and returns the cluster object.
func CreateCoreCluster(cl client.Client, clusterName, infraClusterKind string) *clusterv1.Cluster {
	By("Creating core cluster")

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: CAPINamespace,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       infraClusterKind,
				Name:       clusterName,
				Namespace:  CAPINamespace,
			},
		},
	}
	// TODO(damdo): is there a way to avoid doing this in the generic framework?
	if infraClusterKind == "VSphereCluster" {
		host, port, err := GetControlPlaneHostAndPort(cl)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

		cluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: host,
			Port: port,
		}
	}

	if err := cl.Create(ctx, cluster); err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred())
	}

	Eventually(func() (bool, error) {
		patchedCluster := &clusterv1.Cluster{}
		err := cl.Get(ctx, client.ObjectKeyFromObject(cluster), patchedCluster)
		if err != nil {
			return false, err
		}

		return conditions.IsTrue(patchedCluster, clusterv1.ControlPlaneInitializedCondition), nil
	}, WaitMedium).Should(BeTrue())

	return cluster
}
