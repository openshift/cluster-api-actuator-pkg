module github.com/openshift/cluster-api-actuator-pkg

go 1.13

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/openshift/api v0.0.0-20200526144822-34f54f12813a
	github.com/openshift/cluster-api-provider-gcp v0.0.0
	github.com/openshift/cluster-autoscaler-operator v0.0.0-20190627103136-350eb7249737
	github.com/openshift/library-go v0.0.0-20200512120242-21a1ff978534
	github.com/openshift/machine-api-operator v0.2.1-0.20200611014855-9a69f85c32dd
	github.com/prometheus/common v0.7.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/appengine v1.6.4 // indirect
	k8s.io/api v0.18.3
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v0.18.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/cluster-api-provider-aws v0.0.0-00010101000000-000000000000
	sigs.k8s.io/cluster-api-provider-azure v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.6.0
)

// Use openshift forks
replace sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200618031251-e16dd65fdd85

replace sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200620092221-ff90663025f1
