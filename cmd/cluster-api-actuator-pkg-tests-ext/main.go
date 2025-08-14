package main

import (
	"fmt"
	"os"
	"time"

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
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
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

	ext.AddSuite(e.Suite{
		Name:       "mapi/e2e",
		Qualifiers: []string{`(labels.exists(l, l == "mapi")||labels.exists(l, l == "machine-approver")||labels.exists(l, l == "machine-health-check")) && !labels.exists(l, l == "qe-only")`},
	})

	ext.AddSuite(e.Suite{
		Name:       "capi/e2e",
		Qualifiers: []string{`labels.exists(l, l == "capi") && !labels.exists(l, l == "qe-only")`},
	})

	specs, err := g.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite()
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ginkgo: %+v", err.Error()))
	}

	// Configure platform-specific timeouts before running tests
	specs.AddBeforeAll(func() {
		client, err := framework.LoadClient()
		if err != nil {
			panic(fmt.Sprintf("Failed to load client: %v", err))
		}

		ctx := framework.GetContext()

		platform, err := framework.GetPlatform(ctx, client)
		if err != nil {
			panic(fmt.Sprintf("Failed to get platform: %v", err))
		}

		// Extend timeouts for slower providers
		switch platform {
		case osconfigv1.AzurePlatformType, osconfigv1.GCPPlatformType, osconfigv1.VSpherePlatformType, osconfigv1.OpenStackPlatformType, osconfigv1.PowerVSPlatformType, osconfigv1.NutanixPlatformType:
			framework.WaitShort = 2 * time.Minute  // Normally 1m
			framework.WaitMedium = 6 * time.Minute // Normally 3m
			framework.WaitLong = 30 * time.Minute  // Normally 15m
		}
	})

	ext.AddSpecs(specs)
	extensionRegistry.Register(ext)

	root := &cobra.Command{
		Long: "cluster-api-actuator-pkg tests extension for OpenShift",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(extensionRegistry)...)

	if err := func() error {
		return root.Execute()
	}(); err != nil {
		os.Exit(1)
	}
}
