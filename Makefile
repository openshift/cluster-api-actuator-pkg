BUILD_DEST ?= bin/cluster-api-e2e

GO111MODULE = on
export GO111MODULE
GOFLAGS ?= -mod=vendor
export GOFLAGS
GOPROXY ?=
export GOPROXY

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
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
	  -v "$(PWD)":/go/src/github.com/openshift/cluster-api-actuator-pkg:Z \
	  -w /go/src/github.com/openshift/cluster-api-actuator-pkg \
	  -e "GO111MODULE=$(GO111MODULE)" \
	  -e "GOFLAGS=$(GOFLAGS)" \
	  -e "GOPROXY=$(GOPROXY)" \
	  registry.ci.openshift.org/openshift/release:golang-1.18
  IMAGE_BUILD_CMD = $(ENGINE) build
endif

.PHONY: all
all: check

.PHONY: vendor
vendor:
	$(DOCKER_CMD) ./hack/go-mod.sh

.PHONY: check
check: fmt vet #lint ## Check your code

.PHONY: lint
lint: ## Go lint your code
	( GOLANGCI_LINT_CACHE=$(PROJECT_DIR)/.cache $(GOLANGCI_LINT) run --timeout 10m )

.PHONY: fmt
fmt: ## Go fmt your code
	$(DOCKER_CMD) hack/go-fmt.sh .

.PHONY: goimports
goimports: ## Go fmt your code
	$(DOCKER_CMD) hack/goimports.sh .

.PHONY: vet
vet: ## Apply go vet to all go files
	$(DOCKER_CMD) hack/go-vet.sh ./...

.PHONY: build-e2e
build-e2e:
	$(DOCKER_CMD) go test -c -o "$(BUILD_DEST)" github.com/openshift/cluster-api-actuator-pkg/pkg/

.PHONY: test-e2e
test-e2e: ## Run openshift specific e2e test
	hack/ci-integration.sh $(GINKGO_ARGS) --label-filter='!periodic' -p

.PHONY: test-e2e-periodic
test-e2e-periodic: ## Run openshift specific periodic e2e test
	hack/ci-integration.sh $(GINKGO_ARGS) --label-filter=periodic -p

.PHONY: help
help:
	@grep -E '^[a-zA-Z/0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
