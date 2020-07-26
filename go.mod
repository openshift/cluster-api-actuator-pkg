module github.com/openshift/cluster-api-actuator-pkg

go 1.13

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/google/uuid v1.1.1
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/openshift/api v0.0.0-20200526144822-34f54f12813a
	github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c // indirect
	github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20200701112720-3a7d727c9a10
	github.com/openshift/cluster-autoscaler-operator v0.0.0-20190627103136-350eb7249737
	github.com/openshift/library-go v0.0.0-20200512120242-21a1ff978534
	github.com/openshift/machine-api-operator v0.2.1-0.20200716183840-c9a1f7584b4b
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v0.18.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/cluster-api-provider-aws v0.0.0-00010101000000-000000000000
	sigs.k8s.io/cluster-api-provider-azure v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.6.0
)

// Use openshift forks
replace sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200618031251-e16dd65fdd85

replace sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200620092221-ff90663025f1
