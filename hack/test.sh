#!/bin/bash

set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE}")/..

OPENSHIFT_CI=${OPENSHIFT_CI:-""}
ARTIFACT_DIR=${ARTIFACT_DIR:-""}
GINKGO=${GINKGO:-"go run ${REPO_ROOT}/vendor/github.com/onsi/ginkgo/v2/ginkgo"}
GINKGO_ARGS=${GINKGO_ARGS:-"-v --randomize-all --randomize-suites --keep-going --race --trace --timeout=5m"}
GINKGO_EXTRA_ARGS=${GINKGO_EXTRA_ARGS:-""}

# Ensure that some home var is set and that it's not the root.
# This is required for the kubebuilder cache.
export HOME=${HOME:=/tmp/kubebuilder-testing}
if [ $HOME == "/" ]; then
  export HOME=/tmp/kubebuilder-testing
fi

if [ "$OPENSHIFT_CI" == "true" ] && [ -n "$ARTIFACT_DIR" ] && [ -d "$ARTIFACT_DIR" ]; then # detect ci environment there
  GINKGO_ARGS="${GINKGO_ARGS} --junit-report=junit_cluster-api-actuator-testutils.xml --cover --coverprofile=test-unit-coverage.out --output-dir=${ARTIFACT_DIR}"
fi

# Run only test unit tests
TEST_PACKAGES=$(go list -f "{{ .Dir }}" ./... | grep cluster-api-actuator-pkg/pkg/testutils)

# Print the command we are going to run as Make would.
echo ${GINKGO} ${GINKGO_ARGS} ${GINKGO_EXTRA_ARGS} "<omitted>"
${GINKGO} ${GINKGO_ARGS} ${GINKGO_EXTRA_ARGS} ${TEST_PACKAGES}
# Capture the test result to exit on error after coverage.
TEST_RESULT=$?

if [ -f "${ARTIFACT_DIR}/test-unit-coverage.out" ]; then
  # Convert the coverage to html for spyglass.
  go tool cover -html=${ARTIFACT_DIR}/test-unit-coverage.out -o ${ARTIFACT_DIR}/test-unit-coverage.html

  # Report the coverage at the end of the test output.
  echo -n "Coverage "
  go tool cover -func=${ARTIFACT_DIR}/test-unit-coverage.out | tail -n 1
  # Blank new line after the coverage output to make it easier to read when there is an error.
  echo
fi

# Ensure we exit based on the test result, coverage results are supplementary.
exit ${TEST_RESULT}
