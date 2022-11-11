#!/bin/sh

go run ./vendor/github.com/onsi/ginkgo/v2/ginkgo \
    -timeout 90m \
    -v \
    --fail-fast \
    --no-color \
    "$@" \
    ./pkg/ -- --alsologtostderr -v 4 -kubeconfig ${KUBECONFIG:-~/.kube/config}
