#!/bin/bash

OUTPUT_DIR=${JUNIT_DIR:-"$(pwd)/_out"}
REPORT_POSTFIX="$(date +%s)_${RANDOM}"

go run ./vendor/github.com/onsi/ginkgo/v2/ginkgo \
    -v \
    --timeout=90m \
    --grace-period=5m \
    --fail-fast \
    --no-color \
    --junit-report="junit_cluster_api_actuator_pkg_e2e.xml" \
    --output-dir="${OUTPUT_DIR}" \
    "$@" \
    ./pkg/ -- --alsologtostderr -v 4 -kubeconfig ${KUBECONFIG:-~/.kube/config}
