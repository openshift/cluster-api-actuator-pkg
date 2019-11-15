#!/bin/sh

go test -timeout 90m \
  -v ./pkg/e2e \
  -kubeconfig ${KUBECONFIG:-~/.kube/config} \
  -args -v 5 -logtostderr \
  "$@"
