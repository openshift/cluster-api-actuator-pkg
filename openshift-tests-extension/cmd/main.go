package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	"github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
	"github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"
	"github.com/spf13/cobra"

	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg"
)

var componentKind = "non-payload"

func newRootCommand() (*cobra.Command, error) {
	registry := extension.NewRegistry()
	ext := extension.NewExtension("openshift", componentKind, "cluster-api-actuator-pkg")
	ext.Source.SourceURL = "https://github.com/openshift/cluster-api-actuator-pkg"

	// These suites deliberately retain the existing Makefile's selection rules.
	// Serialize initially: each selection includes serial and disruptive specs.
	for _, suite := range []struct {
		name      string
		qualifier string
	}{
		{"standard", `!labels.exists(l, l == "periodic") && !labels.exists(l, l == "qe-only") && !labels.exists(l, l == "autoscaler")`},
		{"periodic", `labels.exists(l, l == "periodic") && !labels.exists(l, l == "qe-only") && !labels.exists(l, l == "autoscaler")`},
		{"autoscaler", `!labels.exists(l, l == "periodic") && labels.exists(l, l == "autoscaler")`},
		{"periodic-autoscaler", `labels.exists(l, l == "periodic") && labels.exists(l, l == "autoscaler")`},
	} {
		timeout := 3 * time.Hour
		ext.AddSuite(extension.Suite{
			Name:             "openshift/cluster-api-actuator/" + suite.name,
			Qualifiers:       []string{suite.qualifier},
			Parallelism:      1,
			TestTimeout:      &timeout,
			ClusterStability: extension.ClusterStabilityDisruptive,
		})
	}

	// Locate the imported suite's sources, independently of the working directory
	// or checkout name. Dependency-registered specs must not leak into this suite.
	suitePC := reflect.ValueOf(e2e.InitializeSuite).Pointer()
	suiteFile, _ := runtime.FuncForPC(suitePC).FileLine(suitePC)
	sourcePrefix := filepath.Dir(suiteFile) + string(filepath.Separator)

	specs, err := ginkgo.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite(func(spec *et.ExtensionTestSpec) bool {
		for _, location := range spec.CodeLocations {
			if strings.HasPrefix(location, sourcePrefix) && !strings.Contains(location, "/vendor/") {
				return true
			}
		}

		return false
	})
	if err != nil {
		return nil, fmt.Errorf("build actuator Ginkgo specs: %w", err)
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no actuator Ginkgo specs found under %s", sourcePrefix)
	}

	var (
		setupOnce sync.Once
		setupErr  error
	)

	specs.Walk(func(spec *et.ExtensionTestSpec) {
		// Keep generated metadata independent of the checkout/build directory.
		for i, location := range spec.CodeLocations {
			if strings.HasPrefix(location, sourcePrefix) {
				spec.CodeLocations[i] = "pkg/" + strings.TrimPrefix(location, sourcePrefix)
			} else if _, relative, found := strings.Cut(location, "/vendor/"); found {
				spec.CodeLocations[i] = "vendor/" + relative
			}
		}

		run := spec.Run
		spec.Run = func(ctx context.Context) *et.ExtensionTestResult {
			setupOnce.Do(func() { setupErr = e2e.InitializeSuite() })

			if setupErr != nil {
				return &et.ExtensionTestResult{Name: spec.Name, Result: et.ResultFailed, Error: setupErr.Error()}
			}

			return run(ctx)
		}
	})
	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Use:   "cluster-api-actuator-tests-ext",
		Short: "Cluster API actuator tests extension for OpenShift",
	}
	root.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	return root, nil
}

func main() {
	root, err := newRootCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
