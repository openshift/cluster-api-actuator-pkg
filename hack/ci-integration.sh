#!/bin/sh

go run ./vendor/github.com/onsi/ginkgo/ginkgo \
    -timeout 90m \
    -stream \
    -v \
    --failFast \
    --noColor \
    "$@" \
    ./pkg/ -- --alsologtostderr -v 4 -kubeconfig ${KUBECONFIG:-~/.kube/config}
