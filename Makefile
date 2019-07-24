.PHONY: all
all: check

NO_DOCKER ?= 0
ifeq ($(NO_DOCKER), 1)
  DOCKER_CMD =
  IMAGE_BUILD_CMD = imagebuilder
  CGO_ENABLED = 1
  GO = GO111MODULE=on GOFLAGS=-mod=vendor go
else
  DOCKER_CMD := docker run --rm -v "$(PWD)":/cluster-api-actuator-pkg:Z -w /cluster-api-actuator-pkg -e GO111MODULE=on -e GOFLAGS=-mod=vendor openshift/origin-release:golang-1.12
  IMAGE_BUILD_CMD = docker build
  GO = go
endif

.PHONY: depend-update
depend-update:
	go get -u
	go mod vendor

.PHONY: check
check: fmt vet lint test ## Check your code

.PHONY: test
test: # Run unit test
	$(DOCKER_CMD) $(GO) test -race -cover `$(GO) list ./... | grep -v github.com/openshift/cluster-api-actuator-pkg/pkg/e2e`

.PHONY: lint
lint: ## Go lint your code
	hack/go-lint.sh -min_confidence 0.3 $($(GO) list -f '{{ .ImportPath }}' ./...)

.PHONY: fmt
fmt: ## Go fmt your code
	hack/go-fmt.sh .

.PHONY: vet
vet: ## Apply go vet to all go files
	hack/go-vet.sh ./...

.PHONY: build-e2e
build-e2e:
	$(GO) test -c -o bin/e2e github.com/openshift/cluster-api-actuator-pkg/pkg/e2e

.PHONY: test-e2e
test-e2e: ## Run openshift specific e2e test
	# Run operator tests first to preserve logs for troubleshooting test
	# failures and flakes.
	# Feature:Operator tests remove deployments. Thus loosing all the logs
	# previously acquired.
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "Feature:Operators"
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.skip "Feature:Operators|TechPreview"

test-e2e-tech-preview:
	hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "TechPreview"

.PHONY: k8s-e2e
k8s-e2e: ## Run k8s specific e2e test
	# Run operator tests first to preserve logs for troubleshooting test
	# failures and flakes.
	# Feature:Operator tests remove deployments. Thus loosing all the logs
	# previously acquired.
	NAMESPACE=kube-system hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.focus "Feature:Operators"
	NAMESPACE=kube-system hack/ci-integration.sh -ginkgo.v -ginkgo.noColor=true -ginkgo.skip "Feature:Operators|TechPreview"

.PHONY: help
help:
	@grep -E '^[a-zA-Z/0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
