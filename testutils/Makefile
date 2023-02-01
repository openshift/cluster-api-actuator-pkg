GO111MODULE = on
export GO111MODULE
GOFLAGS ?= -mod=vendor
export GOFLAGS
GOPROXY ?=
export GOPROXY

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.26

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
ENVTEST = go run ${PROJECT_DIR}/vendor/sigs.k8s.io/controller-runtime/tools/setup-envtest
GOLANGCI_LINT = go run ${PROJECT_DIR}/vendor/github.com/golangci/golangci-lint/cmd/golangci-lint

NO_DOCKER ?= 0

ifeq ($(shell command -v podman > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=podman
else ifeq ($(shell command -v docker > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=docker
else
	NO_DOCKER=1
endif

USE_DOCKER ?= 0
ifeq ($(USE_DOCKER), 1)
	ENGINE=docker
endif

ifeq ($(NO_DOCKER), 1)
  DOCKER_CMD =
else
  DOCKER_CMD := $(ENGINE) run \
	  --rm \
	  -v "$(PWD)/..":/go/src/github.com/openshift/cluster-api-actuator-pkg:Z \
	  -w /go/src/github.com/openshift/cluster-api-actuator-pkg/testutils \
	  -e "GO111MODULE=$(GO111MODULE)" \
	  -e "GOFLAGS=$(GOFLAGS)" \
	  -e "GOPROXY=$(GOPROXY)" \
	  registry.ci.openshift.org/openshift/release:golang-1.19
  IMAGE_BUILD_CMD = $(ENGINE) build
endif

.PHONY: all
all: lint unit

.PHONY: vendor
vendor: ## Update vendor directory
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: lint
lint: ## Go lint your code
	( GOLANGCI_LINT_CACHE=$(PROJECT_DIR)/.cache $(GOLANGCI_LINT) run --config ../.golangci.yaml --timeout 10m )

.PHONY: unit
unit: ## Run unit tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(PROJECT_DIR)/bin)" ./hack/test.sh

.PHONY: help
help:
	@grep -E '^[a-zA-Z/0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
