FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
WORKDIR /go/src/github.com/openshift/cluster-api-actuator-pkg
COPY . .
RUN go build -o bin/test-server github.com/openshift/cluster-api-actuator-pkg/images/nettest

FROM docker.io/gofed/base:baseci
EXPOSE 8080
COPY --from=builder /go/src/github.com/openshift/cluster-api-actuator-pkg/bin/test-server /
ENTRYPOINT ["/test-server"]
