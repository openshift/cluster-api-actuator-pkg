package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	g "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"

	. "github.com/onsi/ginkgo/v2"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"

	osconfigv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	caov1alpha1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	// If using ginkgo, import your tests here.
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/annotations"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/autoscaler"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/capi"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/infra"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/machinehealthcheck"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/mapi"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/operators"
	_ "github.com/openshift/cluster-api-actuator-pkg/pkg/providers"
)

func init() {
	klog.InitFlags(nil)
	klog.SetOutput(GinkgoWriter)

	if err := machinev1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := caov1alpha1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := osconfigv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := clusterv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := azurev1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := gcpv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := awsv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}

	if err := gcpv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatal(err)
	}
}

func main() {
	extensionRegistry := e.NewRegistry()
	ext := e.NewExtension("openshift", "payload", "cluster-api-actuator-pkg")
	extensionRegistry.Register(ext)

	ext.AddSuite(
		e.Suite{
			Name:       "cluster-infrastructure/test-e2e",
			Qualifiers: []string{`!labels.exists(l, l == "periodic") && !labels.exists(l, l == "qe-only")`},
		})

	ext.AddSuite(
		e.Suite{
			Name:       "cluster-infrastructure/test-e2e-periodic",
			Qualifiers: []string{`labels.exists(l, l == "periodic") && !labels.exists(l, l == "qe-only")`},
		})

	ext.AddSuite(
		e.Suite{
			Name:       "cluster-infrastructure/disruptive",
			Parents:    []string{"openshift/disruptive"},
			Qualifiers: []string{`!labels.exists(l, l == "qe-only")`},
		})

	// Build our specs from ginkgo
	specs, err := g.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite()
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ginkgo: %+v", err.Error()))
	}

	ext.AddSpecs(specs)

	root := &cobra.Command{
		Long: "Cluster Infrastructure cluster-api-actuator-pkg Tests Extension for OpenShift",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(extensionRegistry)...)

	if err := func() error {
		return root.Execute()
	}(); err != nil {
		os.Exit(1)
	}
}
