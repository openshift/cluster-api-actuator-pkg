package framework

import "github.com/onsi/ginkgo/v2"

var (
	LabelDisruptive            = ginkgo.Label("disruptive")
	LabelAutoscaler            = ginkgo.Label("autoscaler")
	LabelOperators             = ginkgo.Label("operators")
	LabelPeriodic              = ginkgo.Label("periodic")
	LabelSpot                  = ginkgo.Label("spot-instances")
	LabelMachines              = ginkgo.Label("machines")
	LabelMachineHealthChecks   = ginkgo.Label("machine-health-checks")
	LabelCloudProviderSpecific = ginkgo.Label("cloud-provider-specific")
	LabelProviderAWS           = ginkgo.Label("AWS")
)
