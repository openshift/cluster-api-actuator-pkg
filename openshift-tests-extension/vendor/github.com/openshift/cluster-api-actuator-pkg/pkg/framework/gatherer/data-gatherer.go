package gatherer

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"

	"k8s.io/klog"
)

const (
	mapiMachines            = "machines.machine.openshift.io"
	mapiMachinesets         = "machinesets.machine.openshift.io"
	controlPlaneMachinesets = "controlplanemachinesets.machine.openshift.io"
	machineHealthChecks     = "machinehealthchecks.machine.openshift.io"

	machineApproverNamespace = "openshift-cluster-machine-approver"
)

var namespacedResourceToGather = []string{
	mapiMachinesets, mapiMachines, controlPlaneMachinesets, machineHealthChecks, "pods",
}

var clusterResourceToGather = []string{"nodes", "clusterautoscaler", "machineautoscaler"}

// StateGatherer MAPI specific wrapper for the CLI helper
// Intended to provide helper functions for gathering MAPI specific resources as well as related pod logs.
type StateGatherer struct {
	CLI       *CLI
	sinceTime time.Time

	specReport *ginkgo.SpecReport

	ctx context.Context
}

// NewStateGatherer initializes StateGatherer.
func NewStateGatherer(ctx context.Context, ocCLI *CLI, gatherSinceTime time.Time) *StateGatherer {
	return &StateGatherer{
		CLI:       ocCLI,
		sinceTime: gatherSinceTime,

		ctx: ctx,
	}
}

// GatherResources helper method to collect MAPI-specific resources such as
// Machines, MachineSets, Nodes, Autoscalers, and so on.
// Store files into '%CLI.outputBasePath%/%test_name%/resources'.
func (sg *StateGatherer) GatherResources() error {
	resourcesSubPath := sg.getSubPath("resources")

	for _, resource := range namespacedResourceToGather {
		klog.Infof("gathering %s", resource)

		if _, err := sg.CLI.Run("get").Args(resource, "-o", "wide").WithSubPath(resourcesSubPath).OutputToFile(resource); err != nil {
			klog.Errorf("%s", err.Error())
			return err
		}

		if _, err := sg.CLI.Run("get").Args(resource, "-o", "yaml").WithSubPath(resourcesSubPath).OutputToFile(fmt.Sprintf("%s_full.yaml", resource)); err != nil {
			klog.Errorf("%s", err.Error())
			return err
		}
	}

	for _, resource := range clusterResourceToGather {
		klog.Infof("gathering %s", resource)

		if _, err := sg.CLI.Run("get").WithoutNamespace().Args(resource, "-o", "wide").WithSubPath(resourcesSubPath).OutputToFile(resource); err != nil {
			klog.Errorf("%s", err.Error())
			return err
		}

		if _, err := sg.CLI.Run("get").WithoutNamespace().Args(resource, "-o", "yaml").WithSubPath(resourcesSubPath).OutputToFile(fmt.Sprintf("%s_full.yaml", resource)); err != nil {
			klog.Errorf("%s", err.Error())
			return err
		}
	}

	return nil
}

// GatherPodLogs collects logs from pods in MAPI-related namespaces since a particular time.
// Skips file creation if no new logs since time were found there.
// Store files into '%CLI.outputBasePath%/%test_name%/logs'.
func (sg *StateGatherer) GatherPodLogs() {
	// TODO dmoiseev: Need to figure out how to collect errors from multiply pods/containers properly and do not
	// TODO dmoiseev: stop gathering from another pods/containers.
	// TODO dmoiseev: This does not return error for now.
	logsSubPath := sg.getSubPath("logs")
	sg.CLI.WithSubPath(logsSubPath).DumpPodLogsSinceTime(sg.ctx, sg.sinceTime)
	sg.CLI.WithSubPath(logsSubPath).WithNamespace(machineApproverNamespace).DumpPodLogsSinceTime(sg.ctx, sg.sinceTime)
}

// GatherAll invokes GatherResources and GatherPodLogs subsequently.
func (sg *StateGatherer) GatherAll() error {
	err := sg.GatherResources()
	sg.GatherPodLogs()

	return err
}

func (sg *StateGatherer) getSubPath(subPath string) string {
	if sg.specReport != nil {
		return filepath.Join(sg.specReport.FullText(), subPath)
	}

	return subPath
}

// WithSpecReport sets ginkgo spec report for the StateGatherer instance.
// This test description uses to extract the test name and store relevant data in a respective sub-folder.
func (sg StateGatherer) WithSpecReport(specReport ginkgo.SpecReport) *StateGatherer {
	sg.specReport = &specReport
	return &sg
}

// WithSinceTime sinceTime parameter for the StateGatherer.
// Uses to indicate test start time and collect logs only since then.
func (sg StateGatherer) WithSinceTime(time time.Time) *StateGatherer {
	sg.sinceTime = time
	return &sg
}
