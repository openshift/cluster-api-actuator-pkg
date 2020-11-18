module github.com/openshift/cluster-api-actuator-pkg

go 1.15

require (
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/google/uuid v1.1.2
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v0.0.0-20200916161728-83f0cb093902
	github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20201002065957-9854f7420570
	github.com/openshift/cluster-autoscaler-operator v0.0.1-0.20201110005855-5fe68233960d
	github.com/openshift/library-go v0.0.0-20200917093739-70fa806b210a
	github.com/openshift/machine-api-operator v0.2.1-0.20201002104344-6abfb5440597
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20201027101359-01387209bb0d
	sigs.k8s.io/cluster-api-provider-aws v0.0.0-00010101000000-000000000000
	sigs.k8s.io/cluster-api-provider-azure v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.6.3
)

// Use openshift forks
replace sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201125052318-b85a18cbf338

replace sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200620092221-ff90663025f1
