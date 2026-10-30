# OpenShift Tests Extension (OTE)

This repository exposes its existing Ginkgo tests through a non-payload OTE
binary, alongside the direct Ginkgo entry point. Test bodies are shared; the
existing `make build-e2e` and `make test-e2e*` workflows remain supported.

`pkg/suite.go` imports the test packages and registers schemes without connecting
to a cluster. Both entry points call `InitializeSuite` during execution to load
the cluster platform and configure provider-specific timeouts. OTE discovery does
not call cluster-dependent initialization.

The entry point lives at `openshift-tests-extension/cmd/main.go` in a separate
Go module. OTE requirements are maintained in that module rather than the root
module. All three modules share the repository's Go workspace and unified vendor
directory, so dependency versions are still selected across the workspace.
`make vendor` includes the extension module, and `make lint` checks it explicitly.

## Build and inspect

```sh
make tests-ext-build
./bin/cluster-api-actuator-tests-ext info
./bin/cluster-api-actuator-tests-ext list tests
./bin/cluster-api-actuator-tests-ext list suites
```

These commands do not require a cluster or `KUBECONFIG`. Test listing includes the
existing actuator specs, excluding specs registered by dependencies. The standard
`run-test`, `run-suite`, `update`, and other framework commands are available.
Execution requires access to a cluster through `KUBECONFIG`.

## Suites and execution

| OTE suite | Existing Make target |
| --- | --- |
| `openshift/cluster-api-actuator/standard` | `test-e2e` |
| `openshift/cluster-api-actuator/periodic` | `test-e2e-periodic` |
| `openshift/cluster-api-actuator/autoscaler` | `test-e2e-autoscaler` |
| `openshift/cluster-api-actuator/periodic-autoscaler` | `test-e2e-periodic-autoscaler` |

Each suite preserves its Make target's `periodic`, `autoscaler`, and `qe-only`
selection rules. In particular, autoscaler selections retain the existing behavior
of not excluding `qe-only`. Other labels and runtime platform skips are unchanged.

These selections mix serial, ordered, and disruptive tests. All four OTE suites
therefore use parallelism one and disruptive cluster stability, with a
three-hour per-test timeout. They do not advertise generic conformance parents.
Ordered tests are executed individually by the adapter, including their associated
setup and cleanup; cross-test shared state must not be assumed in this path.

```sh
KUBECONFIG=/path/to/kubeconfig ./bin/cluster-api-actuator-tests-ext \
    run-suite openshift/cluster-api-actuator/standard

KUBECONFIG=/path/to/kubeconfig ./bin/cluster-api-actuator-tests-ext \
    run-test '<exact test name from list tests>'
```

The adapter requires the OpenShift Ginkgo fork. Its pinned revision comes from
the `v2.32.0-openshift-5.1` branch. The replacement participates in the shared workspace, so both OTE and
the direct Ginkgo workflow use that fork. OTE initializes cluster defaults once per
process and reports initialization failures as failed test results.

The OTE dependency is pinned to
`v0.0.0-20260812190735-a8b44e9f9fed`, matching the version used by the
`openshift/origin` runner on `release-5.1` at the time of integration. The Ginkgo
fork is pinned independently to support this repository's test suite.

## Component identity

The extension identity is `openshift:non-payload:cluster-api-actuator-pkg`.
This repository provides a test suite, not a payload component. The extension is
packaged in the `cluster-api-actuator-pkg-test` CI image, outside the OpenShift
release payload.

The identity does not itself register the extension with `openshift-tests`;
discovery requires the cluster resources described under
[Non-payload discovery](#non-payload-discovery). No payload image or payload
extension registry entry is required.

## Packaging

```sh
make tests-ext-package
```

This produces both `bin/cluster-api-actuator-tests-ext` and
`bin/cluster-api-actuator-tests-ext.gz`. The executable is built with
`CGO_ENABLED=0` and `GO_COMPLIANCE_POLICY=exempt_all`. Gzip omits the original
filename and timestamp, making compression reproducible for identical binaries.
The existing `Dockerfile.ci` builds the extension and installs the compressed
artifact at `/usr/bin/cluster-api-actuator-tests-ext.gz`, preserving the existing
builder environment and source checkout. No separate runtime image is introduced.

Build and inspect the existing test image with:

```sh
podman build -f Dockerfile.ci -t cluster-api-actuator-pkg-test:ote .
podman run --rm cluster-api-actuator-pkg-test:ote \
    sh -c 'gzip -dc /usr/bin/cluster-api-actuator-tests-ext.gz > /tmp/tests-ext && chmod +x /tmp/tests-ext && /tmp/tests-ext info && /tmp/tests-ext list tests'
```

The release configuration already builds this Dockerfile as
`cluster-api-actuator-pkg-test`. That test image is intentionally excluded from
payload promotion. The separately promoted `cluster-api-actuator-pkg` image comes
from the CI `src` image, not this Dockerfile, and is not the carrier for this
extension. Embedding the binary in the CI test image provides packaging, but does
not by itself enable non-payload discovery by `openshift-tests`.

## Test metadata

```sh
make tests-ext-update
make tests-ext-verify
```

Metadata is generated by OTE under `hack/.openshift-tests-extension/`. Commit
metadata updates together with extension changes. Verification copies committed
metadata into a temporary directory, runs the OTE update command there, and
compares the result without modifying the checkout. Source locations are
normalized to repository-relative paths so different checkout directories do not
change the metadata. Missing metadata, stale metadata, and unacknowledged test
removals cause verification to fail.

## Validation

```sh
make tests-ext-verify
file bin/cluster-api-actuator-tests-ext
gzip -t bin/cluster-api-actuator-tests-ext.gz
```

## Non-payload discovery

`openshift-tests` discovers non-payload extensions through ImageStreamTags on the
cluster under test. Discovery requires a runner with non-payload extension support
and the following cluster resources:

- An ImageStreamTag referencing the extension image, with these annotations:
  - `testextension.redhat.io/component: cluster-api-actuator-pkg`
  - `testextension.redhat.io/binary: /usr/bin/cluster-api-actuator-tests-ext.gz`
- The `TestExtensionAdmission` CRD and a cluster-scoped `TestExtensionAdmission`
  resource permitting the ImageStream's `namespace/name`.

The runner and cluster must have access to pull the referenced image. The image
reference determines which build supplies the tests; the component identity does
not select an image version. These cluster resources are managed outside this
repository and are not created by building or invoking the extension directly.

With discovery configured, `openshift-tests list extensions -o json` lists the
extension. The suites above can be executed through `openshift-tests run`, for
example:

```sh
openshift-tests run openshift/cluster-api-actuator/standard \
    --junit-dir=/path/to/artifacts/junit
```

The extension's suites have no parent-suite registrations, so running a generic
conformance suite does not automatically select these tests.
