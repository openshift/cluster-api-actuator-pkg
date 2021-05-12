module github.com/openshift/cluster-api-actuator-pkg

go 1.15

require (
	github.com/google/uuid v1.1.2
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/openshift/api v0.0.0-20210505180709-d0a89da74761
	github.com/openshift/cluster-api-provider-gcp v0.0.1-0.20210507003447-984c7a004201
	github.com/openshift/cluster-autoscaler-operator v0.0.1-0.20210429051843-6c36bbd942c0
	github.com/openshift/library-go v0.0.0-20210408164723-7a65fdb398e2
	github.com/openshift/machine-api-operator v0.2.1-0.20210512152848-6afce911d47e
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/cluster-api-provider-aws v0.0.0-00010101000000-000000000000
	sigs.k8s.io/cluster-api-provider-azure v0.0.0-00010101000000-000000000000
	sigs.k8s.io/controller-runtime v0.9.0-beta.1.0.20210512131817-ce2f0c92d77e
)

// Use openshift forks
replace sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20210505150511-f9cb840ae412

replace sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20210505133115-b2eda16dd665
