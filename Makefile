GO111MODULE = on
export GO111MODULE
GOFLAGS += -mod=vendor
export GOFLAGS
GOPROXY ?=
export GOPROXY

NO_DOCKER ?= 0
ifeq ($(NO_DOCKER), 1)
  DOCKER_CMD =
  IMAGE_BUILD_CMD = imagebuilder
else
  DOCKER_CMD := docker run \
	  --rm \
	  -v "$(PWD)":/go/src/github.com/openshift/cluster-api-actuator-pkg:Z \
	  -w /go/src/github.com/openshift/cluster-api-actuator-pkg \
	  -e "GO111MODULE=$(GO111MODULE)" \
	  -e "GOFLAGS=$(GOFLAGS)" \
	  -e "GOPROXY=$(GOPROXY)" \
	  openshift/origin-release:golang-1.12
  IMAGE_BUILD_CMD = docker build
endif

.PHONY: all
all: check

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: check
check: fmt vet lint test ## Check your code

.PHONY: test
test: # Run unit test
	$(DOCKER_CMD) go test -race -cover $$(go list ./... | grep -v github.com/openshift/cluster-api-actuator-pkg/pkg/e2e)

.PHONY: lint
lint: ## Go lint your code
	# TODO(spangenberg): This thing was never working beacuse it was using $ instead of $$
	# Fixing it causes CI to fail, this will be handles in a seperate PR.
	# hack/go-lint.sh -min_confidence 0.3 $$(go list ./...)

.PHONY: fmt
fmt: ## Go fmt your code
	hack/go-fmt.sh .

.PHONY: vet
vet: ## Apply go vet to all go files
	hack/go-vet.sh ./...

.PHONY: build-e2e
build-e2e:
	go test -c -o bin/e2e github.com/openshift/cluster-api-actuator-pkg/pkg/e2e

.PHONY: test-e2e
test-e2e: ## Run openshift specific e2e test
	# Run operator tests first to preserve logs for troubleshooting test
	# failures and flakes.
	# Feature:Operator tests remove deployments. Thus loosing all the logs
	# previously acquired.
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "Feature:Operators" -ginkgo.failFast
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.skip "Feature:Operators|TechPreview" -ginkgo.failFast

test-e2e-tech-preview:
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "TechPreview" -ginkgo.failFast

.PHONY: k8s-e2e
k8s-e2e: ## Run k8s specific e2e test
	# Run operator tests first to preserve logs for troubleshooting test
	# failures and flakes.
	# Feature:Operator tests remove deployments. Thus loosing all the logs
	# previously acquired.
	NAMESPACE=kube-system hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "Feature:Operators" -ginkgo.failFast
	NAMESPACE=kube-system hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.skip "Feature:Operators|TechPreview" -ginkgo.failFast

.PHONY: help
help:
	@grep -E '^[a-zA-Z/0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
