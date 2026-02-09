# AGENTS.md

> E2E testing framework and utilities for Machine API and Cluster API in OpenShift.

## Overview

**cluster-api-actuator-pkg** is a shared Go library providing E2E testing frameworks and utilities for **Machine API** and **Cluster API** in OpenShift.

**Primary Components:**
- **E2E testing framework** (`pkg/framework/`) - Core test infrastructure for Machine/MachineSet resources
- **Test suites** - Machine API operators, CAPI resources, MachineHealthCheck, webhooks, lifecycle hooks
- **Test utilities** (`testutils/resourcebuilder/`) - Builder pattern for creating test resources
- **Cloud provider integrations** - AWS, Azure, GCP, OpenStack, PowerVS, Nutanix, vSphere

## Quick Reference

| What | Where |
|------|-------|
| **Language** | Go 1.24.0 |
| **Test Framework** | Ginkgo v2 + Gomega |
| **Main Namespaces** | `openshift-machine-api`, `openshift-cluster-api` |
| **Build & Test** | `make help` for all targets |
| **Dependencies** | See `go.mod` |
| **Testing Guide** | `README.md`, `docs/tests.md` |
| **Linter Config** | `.golangci.yaml` |

## Do's and Don'ts

### ✅ DO

**Code:**
- Read existing code BEFORE making changes
- Use `testutils/resourcebuilder` for creating test resources
- Use `klog` for logging (never `fmt.Printf` or `log`)
- Wrap errors: `fmt.Errorf("context: %w", err)`
- Group imports: stdlib → third-party → OpenShift → local
- Run `make lint` before committing

**Testing:**
- Use Ginkgo `Describe`/`Context`/`It` blocks
- Add test labels: `[Slow]`, `[Serial]`, `[Feature:Machines]`
- Use dot imports ONLY in `*_test.go` files
- Test against real OpenShift clusters

**Cloud Providers:**
- Maintain feature parity across all providers
- Check all provider implementations when adding features

### ❌ DON'T

- Don't commit without `make lint` passing
- Don't manually create structs - use builders
- Don't use dot imports in non-test files
- Don't modify vendored dependencies
- Don't return naked errors
- Don't hardcode timeouts - use framework constants
- Don't commit credentials or secrets
- Don't log sensitive data

## Project Structure

```
cluster-api-actuator-pkg/
├── pkg/
│   ├── framework/          ⭐ Core E2E testing framework
│   │   ├── framework.go    # Constants, client setup
│   │   ├── machines.go     # Machine helpers
│   │   ├── machinesets.go  # MachineSet helpers
│   │   ├── capi_*.go       # CAPI-specific helpers
│   │   └── gatherer/       # Debug data collection
│   │
│   ├── capi/               ⭐ Cluster API E2E tests
│   │   ├── aws.go          # AWS CAPI MachineSet tests
│   │   ├── azure.go        # Azure CAPI MachineSet tests
│   │   ├── gcp.go          # GCP CAPI MachineSet tests
│   │   └── core.go         # Core CAPI tests
│   │
│   ├── mapi/               ⭐ Machine API E2E tests
│   │   └── gcp.go          # GCP MAPI MachineSet tests
│   │
│   ├── operators/          ⭐ Operator E2E tests
│   │   ├── machine-api-operator.go      # MAO tests
│   │   ├── cluster-machine-approver.go  # CMA tests
│   │   └── cluster-autoscaler-operator.go
│   │
│   ├── machinehealthcheck/ ⭐ MachineHealthCheck E2E tests
│   ├── infra/              ⭐ Infrastructure E2E tests (spot, lifecycle hooks, webhooks)
│   ├── providers/          # Cloud provider implementations
│   └── annotations/        # Cloud provider annotations
│
├── testutils/
│   └── resourcebuilder/    ⭐ Builder pattern for test resources
│       ├── cluster-api/    # CAPI builders
│       ├── machine/        # Machine API builders
│       └── core/           # K8s core builders
│
├── docs/
│   ├── tests.md            # All test descriptions
│   └── cluster-wide-proxy.md
│
├── hack/
│   └── ci-integration.sh   # E2E test runner
│
├── Makefile                # Build targets
└── .golangci.yaml          # Linter config
```

**Key Files:**
- Constants: `pkg/framework/framework.go:30-50`
- Framework helpers: `pkg/framework/`
- E2E test suites: `pkg/capi/`, `pkg/mapi/`, `pkg/operators/`, `pkg/machinehealthcheck/`, `pkg/infra/`
- Test resource builders: `testutils/resourcebuilder/`

## Code Examples

### ✅ GOOD: Resource Builders

```go
import "github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder"

// Use builders for test resources
machine := resourcebuilder.Machine().
    WithNamespace("openshift-machine-api").
    WithName("test-worker-0").
    Build()

machineSet := resourcebuilder.MachineSet().
    WithNamespace("openshift-machine-api").
    WithReplicas(3).
    Build()
```

### ✅ GOOD: Error Handling

```go
func (f *Framework) CreateMachine(ctx context.Context, machine *machinev1.Machine) error {
    if err := f.Client.Create(ctx, machine); err != nil {
        return fmt.Errorf("failed to create machine %s/%s: %w",
            machine.Namespace, machine.Name, err)
    }
    return nil
}
```

### ✅ GOOD: Ginkgo Tests

```go
var _ = Describe("Machines", func() {
    Context("with MachineSet", func() {
        It("should create and delete machines", func(ctx SpecContext) {
            By("Creating a MachineSet")
            // Test implementation
        })

        It("should handle deletion gracefully [Slow]", func(ctx SpecContext) {
            // Test implementation
        })
    })
})
```

### ✅ GOOD: Logging

```go
klog.InfoS("Creating machine", "namespace", machine.Namespace, "name", machine.Name)
klog.ErrorS(err, "Failed to delete machineset", "namespace", ms.Namespace, "name", ms.Name)
```

### ❌ BAD: Manual Struct Creation

```go
// DON'T DO THIS
machine := &machinev1.Machine{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "test-machine",
        Namespace: "openshift-machine-api",
        Labels: map[string]string{
            "machine.openshift.io/cluster-api-cluster": "test-cluster",
        },
    },
    Spec: machinev1.MachineSpec{
        // ... many fields
    },
}

// USE BUILDERS INSTEAD (see above)
```

### ❌ BAD: Wrong Logging

```go
// DON'T DO THIS
fmt.Printf("Creating machine: %s\n", machine.Name)
log.Println("Machine created")

// USE KLOG (see above)
```

### ❌ BAD: Naked Errors

```go
// DON'T DO THIS
if err := client.Create(ctx, machine); err != nil {
    return err
}

// WRAP ERRORS (see above)
```

## Important Constants

From `pkg/framework/framework.go:30-50`:

```go
const (
    // Namespaces - ALWAYS use these
    MachineAPINamespace = "openshift-machine-api"
    ClusterAPINamespace = "openshift-cluster-api"

    // Labels
    WorkerNodeRoleLabel = "node-role.kubernetes.io/worker"
    ClusterKey          = "machine.openshift.io/cluster-api-cluster"
    MachineSetKey       = "machine.openshift.io/cluster-api-machineset"

    // Timeouts
    PollNodesReadyTimeout = 10 * time.Minute
    RetryShort            = 1 * time.Second
    RetryMedium           = 5 * time.Second

    // Machine Phases
    MachinePhaseRunning = "Running"
    MachinePhaseFailed  = "Failed"
)
```

**Don't hardcode these values!**

## Commands

See `Makefile` for all targets. Run `make help` for descriptions.

**Common commands:**
```bash
make lint          # ALWAYS run before committing
make build-e2e     # Build E2E test binaries
make unit          # Run unit tests (testutils only)
make test-e2e      # Run E2E tests
```

**E2E Testing:**

Complete instructions in `README.md`. Quick start:

1. Prerequisites: OpenShift 4 cluster, CVO disabled
2. Build: `make build-e2e`
3. Run: `./hack/ci-integration.sh -focus "Machines should" -v`

**Test labels:**
- `[Slow]` - Takes > 5 minutes
- `[Serial]` - Can't run in parallel
- `[Feature:Machines]` - Machine API tests
- `periodic` - Periodic tests
- `qe-only` - QE-specific tests

See `docs/tests.md` for all 40+ test descriptions.

## Adding Cloud Providers

**Checklist:**
1. [ ] Create `pkg/capi/<provider>.go`
2. [ ] Create `pkg/providers/<provider>.go` if needed
3. [ ] Add builders in `testutils/resourcebuilder/cluster-api/infrastructure/`
4. [ ] Add failure domains in `testutils/resourcebuilder/machine/v1/`
5. [ ] Add annotations in `pkg/annotations/<provider>.go`
6. [ ] Add E2E tests
7. [ ] Add client helpers in `pkg/framework/<provider>_client.go`
8. [ ] Update this AGENTS.md

**Follow existing patterns:**
- AWS: `pkg/capi/aws.go`, `pkg/providers/aws.go`, `pkg/framework/aws_client.go`
- GCP: `pkg/capi/gcp.go`, `pkg/framework/gcp_client.go`
- Azure: `pkg/capi/azure.go`

**Current providers:** AWS, Azure, GCP, OpenStack, PowerVS, Nutanix, vSphere

## Security

**Never commit:**
- AWS keys (`AKIA*`, secret access keys)
- GCP service account JSON
- Azure credentials
- Kubeconfigs with embedded certs/tokens

**Never log:**
- Credentials, tokens, keys

**Use Kubernetes Secrets:**
```go
secretRef := corev1.LocalObjectReference{Name: "aws-credentials"}
```

**Webhook security patterns:** See `pkg/infra/webhooks.go`, `pkg/framework/webhooks.go`

## OpenShift Context

**Namespaces (use constants!):**
- `openshift-machine-api` - Machine API resources
- `openshift-cluster-api` - Cluster API resources

**This library provides E2E tests for:**
- machine-api-operator - Machine lifecycle
- cluster-machine-approver - Node CSR approval

**Platform detection:**
```go
const GlobalInfrastuctureName = "cluster"
// Check Infrastructure.Status.PlatformStatus.Type for: AWS, Azure, GCP, etc.
```

## Development Workflow

1. Understand scope (bug fix / feature / enhancement)
2. Read existing code for patterns
3. Use builders for test resources
4. Write/update E2E tests (unit tests only in testutils)
5. Run `make lint`
6. Test locally against OpenShift cluster (see `README.md`)
7. Consider impact (shared library - breaking changes affect consumers)

**Review checklist:**
- [ ] Follows existing patterns
- [ ] Uses builders from `testutils/resourcebuilder`
- [ ] Uses `klog` for logging
- [ ] Errors wrapped with context
- [ ] Uses constants (no hardcoded values)
- [ ] `make lint` passes
- [ ] Tests added/updated (see `docs/tests.md`)
- [ ] All cloud providers considered
- [ ] No credentials committed
- [ ] Documentation updated if needed

## Troubleshooting

**Linter failures:**
```bash
make lint
# Common issues:
# - Unused imports
# - Undefined constants (check pkg/framework/framework.go)
# - Wrong import order (stdlib → third-party → OpenShift → local)
```

**Test failures:**
```bash
./hack/ci-integration.sh -focus "test name" -v

# Check CVO is disabled (required - see README.md)
oc get deployment -n openshift-cluster-version cluster-version-operator
# Should show 0/0 replicas
```

**Builder issues:**
```go
// Wrong - WithReplicas is on MachineSet, not Machine
machine := resourcebuilder.Machine().WithReplicas(3).Build()

// Right
machineSet := resourcebuilder.MachineSet().WithReplicas(3).Build()
```

## Documentation

- **`README.md`** - E2E testing setup (cluster creation, CVO, execution)
- **`docs/tests.md`** - All 40+ test descriptions with timing/recommendations
- **`docs/cluster-wide-proxy.md`** - Proxy configuration
- **`Makefile`** - Build targets (`make help`)
- **`go.mod`** - Dependencies
- **`OWNERS`** - Maintainers

---

**About:** This file guides AI coding agents. For humans, see README.md and docs/. Follows [AGENTS.md spec](https://agents.md/).

**Principle:** References existing docs instead of duplicating. Check README.md, Makefile, docs/tests.md for latest info.

**Updated:** 2025-11-26
