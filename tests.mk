# Test-specific Makefile
# For running tests with different backend configurations
# Supports 3 test scenarios: pure MAPI, pure CAPI, and conversion layer testing

.PHONY: test-mapi test-capi test-mapi-with-capi-auth test-all show-config help

.DEFAULT_GOAL := help

test-mapi: ## Run MAPI authoritative testing (MAPI auth, CAPI mirrors)
	@echo "Running tests with MAPI backend and MAPI authoritative API (MAPI authoritative, CAPI mirrors)"
	TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI go test -v ./pkg/unified/ -ginkgo.v

test-capi: ## Run pure CAPI testing (no MAPI involvement)
	@echo "Running tests with CAPI backend (pure CAPI, no MAPI involved)"
	TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v

test-mapi-with-capi-auth: ## Run conversion layer testing (CAPI becomes authoritative)
	@echo "Running tests with MAPI backend but CAPI authoritative API (OpenShift conversion layer)"
	TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v

test-all: test-mapi test-capi test-mapi-with-capi-auth ## Run all supported test configurations
	@echo "All test configurations completed"

show-config: ## Show current test configuration
	@echo "Current test configuration:"
	@echo "TEST_BACKEND_TYPE: $(or $(TEST_BACKEND_TYPE),MAPI)"
	@echo "TEST_AUTHORITATIVE_API: $(or $(TEST_AUTHORITATIVE_API),MAPI)"

help: ## Show this help message
	@echo "Unified Test Framework - Makefile for testing MAPI/CAPI backends"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Test scenarios explained:"
	@echo "  1. MAPI + MAPI Auth:        MAPI MachineSets with authoritativeAPI: MachineAPI"
	@echo "                              (MAPI remains authoritative, CAPI mirrors created)"
	@echo "  2. CAPI + CAPI Auth:        Pure CAPI MachineSets (no MAPI involved)"
	@echo "  3. MAPI + CAPI Auth:        MAPI MachineSets with authoritativeAPI: ClusterAPI"
	@echo "                              (CAPI becomes authoritative via conversion layer)"
	@echo ""
	@echo "Environment variables:"
	@echo "  TEST_BACKEND_TYPE           - Set backend type (MAPI or CAPI)"
	@echo "  TEST_AUTHORITATIVE_API      - Set authoritative API type (MAPI or CAPI)"
	@echo ""
	@echo "Examples:"
	@echo "  make -f tests.mk test-mapi                  # Run MAPI authoritative testing"
	@echo "  make -f tests.mk test-all                   # Run all 3 scenarios"
	@echo "  make -f tests.mk show-config                # Check current settings"
