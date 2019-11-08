FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
WORKDIR /go/src/github.com/openshift/cluster-api-actuator-pkg
COPY . .
ENV NO_DOCKER=1
ENV BUILD_DEST=/go/bin/cluster-api-e2e
RUN unset VERSION && GOPROXY=off make build-e2e
CMD ["/go/bin/cluster-api-e2e"]
