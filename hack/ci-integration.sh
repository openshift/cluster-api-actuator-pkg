#!/bin/sh

go run ./vendor/github.com/onsi/ginkgo/ginkgo \
    -timeout 90m \
    -p -stream \
    -v \
    -failFast \
    "$@" \
    ./pkg/e2e/ -- --alsologtostderr -v 4 -kubeconfig ${KUBECONFIG:-~/.kube/config}
