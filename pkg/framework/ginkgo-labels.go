package framework

import "github.com/onsi/ginkgo/v2"

var (
	// LabelAutoscaler applies to tests related to the cluster autoscaler functionality.
	LabelAutoscaler = ginkgo.Label("autoscaler")

	// LabelCAPI applies to tests related to Cluster API (CAPI) functionality.
	LabelCAPI = ginkgo.Label("capi")

	// LabelCCM applies to tests related to the Cloud Controller Manager (CCM).
	LabelCCM = ginkgo.Label("ccm")

	// LabelDevOnly indicates that the test can run in dev account only.
	LabelDevOnly = ginkgo.Label("dev-only")

	// LabelDisruptive marks tests that are disruptive in nature and may affect cluster stability.
	LabelDisruptive = ginkgo.Label("disruptive")

	// LabelLEVEL0 indicates that the test is a basic or critical test, if failed then block release.
	LabelLEVEL0 = ginkgo.Label("LEVEL0")

	// LabelMachineApprover applies to tests for the machine approver functionality.
	LabelMachineApprover = ginkgo.Label("machine-approver")

	// LabelMachineHealthCheck applies to tests for Machine Health Checks (MHC) functionality.
	LabelMachineHealthCheck = ginkgo.Label("machine-health-check")

	// LabelMAPI applies to tests related to the Machine API (MAPI).
	LabelMAPI = ginkgo.Label("mapi")

	// LabelPeriodic marks tests that are meant to run periodically.
	LabelPeriodic = ginkgo.Label("periodic")

	// LabelQEOnly indicates that the test can run in qe account only.
	LabelQEOnly = ginkgo.Label("qe-only")
)
