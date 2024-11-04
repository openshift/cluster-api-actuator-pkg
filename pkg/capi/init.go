package capi

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	ibmpowervsv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	vspherev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	configv1 "github.com/openshift/api/config/v1"
	mapiv1 "github.com/openshift/api/machine/v1beta1"
)

const (
	infrastructureName                                                = "cluster"
	infraAPIVersion                                                   = "infrastructure.cluster.x-k8s.io/v1beta1"
	managedByAnnotationValueClusterCAPIOperatorInfraClusterController = "cluster-capi-operator-infracluster-controller"
)

func init() {
	utilruntime.Must(configv1.Install(scheme.Scheme))
	utilruntime.Must(awsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(gcpv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(azurev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(mapiv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(ibmpowervsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(vspherev1.AddToScheme(scheme.Scheme))
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster API Suite")
}
