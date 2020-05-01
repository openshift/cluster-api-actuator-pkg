module github.com/openshift/cluster-api-actuator-pkg

go 1.13

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.8.1
	github.com/openshift/api v3.9.1-0.20190517100836-d5b34b957e91+incompatible
	github.com/openshift/cluster-autoscaler-operator v0.0.0-20190627103136-350eb7249737
	github.com/openshift/library-go v0.0.0-20200324092245-db2a8546af81
	github.com/openshift/machine-api-operator v0.2.1-0.20200402110321-4f3602b96da3
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/cluster-api-provider-aws v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.6.0
)

// Use openshift forks
replace sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200429204912-67edca9c65de
