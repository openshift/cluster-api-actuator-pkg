# Unified MAPI and CAPI Test Framework

## Overview

This unified test framework provides an abstraction layer that allows the same test suite to run against both Machine API (MAPI) and Cluster API (CAPI) backends without requiring complete reimplementation. The framework supports testing the OpenShift conversion layer functionality, enabling comprehensive validation of API migration scenarios.

## Recent Updates

The unified framework has been enhanced with the following capabilities:

1. **Machine Template Configuration System** - A unified configuration layer (`pkg/unified/config/template_config.go`) that allows specifying machine instance configurations (spot instances, placement groups, tenancy, etc.) in a backend-agnostic way. The same configuration automatically applies to both MAPI `ProviderSpec` and CAPI template specs.

2. **E2E Test Suite** - A dedicated end-to-end test suite (`pkg/unified/e2e/`) demonstrating real-world usage of the framework, including AWS spot instance testing and standard MachineSet lifecycle tests.

3. **Test Helper Utilities** - Reusable test helpers (`pkg/unified/e2e/test_helpers.go`) that simplify common test operations such as creating templates, MachineSets, and validating configurations across different backends.

4. **Improved Package Organization** - Clear separation of concerns with dedicated packages for backends, configuration, and E2E tests.

## Architecture

### Core Components

The unified framework consists of several key components:

1. **MachineBackend Interface** (`pkg/unified/backends/backend_interface.go`) - Abstract interface defining common operations
2. **MAPI Backend Implementation** (`pkg/unified/backends/mapi_backend.go`) - MAPI-specific operations
3. **CAPI Backend Implementation** (`pkg/unified/backends/capi_backend.go`) - CAPI-specific operations
4. **Unified Test Framework** (`pkg/unified/framework.go`) - Orchestrates testing across different backends
5. **Test Configuration Management** (`pkg/unified/config/`) - Environment-based configuration system
6. **Machine Template Configuration** (`pkg/unified/config/template_config.go`) - Unified configuration for machine templates across backends
7. **E2E Test Suite** (`pkg/unified/e2e/`) - End-to-end tests using the unified framework

### Package Structure

```
pkg/unified/
├── framework.go              # Main UnifiedFramework implementation
├── backends/
│   ├── backend_interface.go  # MachineBackend interface definition
│   ├── mapi_backend.go       # MAPI backend implementation
│   ├── capi_backend.go       # CAPI backend implementation
├── config/
│   ├── test_config.go        # Test configuration and environment variables
│   └── template_config.go    # Machine template configuration structs
└── e2e/
    ├── aws_machineset.go     # AWS platform test implementations
    └── test_helpers.go       # Reusable test helper functions
```

### Backend Types and Authoritative APIs

- **Backend Type**: Determines which client/resources the test code uses (MAPI or CAPI)
- **Authoritative API**: Test framework configuration parameter that determines what value to set in the MachineSet's `authoritativeAPI` field

## Configuration

### Environment Variables

Configure the test framework using these environment variables:

```bash
# Set backend type (MAPI or CAPI)
export TEST_BACKEND_TYPE=MAPI

# Set authoritative API type (MAPI or CAPI)
export TEST_AUTHORITATIVE_API=MAPI
```

### Supported Test Scenarios

The framework supports three main testing scenarios:

1. **MAPI Authoritative Testing**
   ```bash
   TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI
   ```
   - Uses MAPI backend to create MAPI MachineSets with `authoritativeAPI: MachineAPI`
   - MAPI serves as the authoritative API for machine lifecycle management
   - OpenShift automatically creates corresponding CAPI mirror resources
   - Machine lifecycle is controlled by MAPI controllers

2. **Pure CAPI Testing**
   ```bash
   TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI
   ```
   - Uses CAPI backend to create CAPI MachineSets
   - CAPI serves as the authoritative API
   - Pure CAPI behavior

3. **CAPI Authoritative Testing**
   ```bash
   TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI
   ```
   - Test code uses MAPI backend to create MAPI MachineSets with `authoritativeAPI: ClusterAPI`
   - CAPI serves as the authoritative API for machine lifecycle management
   - Machine lifecycle management is transferred to CAPI controllers

## Usage

### Using Makefile Targets (Recommended)

The `Makefile.test` provides convenient targets for different test scenarios:

```bash
# MAPI Authoritative testing: MAPI backend + MAPI authoritative
make -f Makefile.test test-mapi

# Pure CAPI testing: CAPI backend + CAPI authoritative
make -f Makefile.test test-capi

# CAPI Authoritative testing: MAPI backend + CAPI authoritative
make -f Makefile.test test-mapi-with-capi-auth

# Run all test configurations
make -f Makefile.test test-all

# Show current configuration
make -f Makefile.test show-config

# Display help
make -f Makefile.test help
```

### Direct go test Usage

```bash
# MAPI Authoritative testing
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=MAPI go test -v ./pkg/unified/ -ginkgo.v

# Pure CAPI testing
TEST_BACKEND_TYPE=CAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v

# CAPI Authoritative testing
TEST_BACKEND_TYPE=MAPI TEST_AUTHORITATIVE_API=CAPI go test -v ./pkg/unified/ -ginkgo.v
```

## Available Test Targets

| Target | Description |
|--------|-------------|
| `test-mapi` | Run tests with MAPI backend and MAPI authoritative API |
| `test-capi` | Run tests with CAPI backend and CAPI authoritative API |
| `test-mapi-with-capi-auth` | Run tests with MAPI backend but CAPI authoritative API |
| `test-all` | Run all supported test configurations |
| `show-config` | Show current test configuration |
| `help` | Display help information |

## Benefits

1. **Code Reuse**: Write tests once, run against multiple backends
2. **Conversion Testing**: Validate OpenShift conversion layer functionality
3. **Flexible Configuration**: Easy switching between test modes
4. **Comprehensive Coverage**: Test both pure API scenarios and conversion layers

## Key Features

### Machine Template Configuration

The framework provides a unified configuration system for machine templates that works across both MAPI and CAPI backends. This is implemented in `pkg/unified/config/template_config.go`.

#### Supported Configurations

**AWS Platform:**
- **Spot Instances**: Configure spot market options with optional max price
- **Placement Groups**: Specify placement group names for instance grouping
- **KMS Keys**: Configure encryption keys
- **Additional Tags**: Add custom tags to instances
- **Tenancy**: Configure instance tenancy (default, dedicated, host)
- **Network Interface Type**: Support for EFA (Elastic Fabric Adapter)
- **Non-Root Volumes**: Configure additional storage volumes

#### Usage Example

```go
// Create spot instance configuration
spotConfig := &config.MachineTemplateConfig{
    AWS: &config.AWSMachineConfig{
        SpotMarketOptions: &config.SpotMarketConfig{
            MaxPrice: nil, // Use default price
        },
        Tenancy: func() *string { s := "default"; return &s }(),
    },
}

// Create template with configuration
tpl, err := uf.CreateMachineTemplate(ctx, cl, platform, backends.BackendMachineTemplateParams{
    Name:     "my-template",
    Platform: platform,
    Spec:     spotConfig,
})
```

The configuration is automatically applied to either MAPI `ProviderSpec` or CAPI `AWSMachineTemplate` based on the backend type.

### Test Helpers

The framework provides test helpers (`pkg/unified/e2e/test_helpers.go`) that simplify common test operations:

- **CreateTemplate**: Creates a machine template with optional configuration
- **CreateMachineSet**: Creates a MachineSet with specified parameters
- **DeleteTemplate/DeleteMachineSet**: Cleanup operations
- **VerifyMachineSetContainsString**: Flexible validation that works across backends
- **SkipIfNotPlatform**: Skip tests for unsupported platforms

## Development Guidelines

### Adding New Tests

1. Use the `UnifiedFramework` interface instead of direct API calls
2. Write backend-agnostic test logic using the provided abstractions
3. Leverage test helpers from `pkg/unified/e2e/test_helpers.go` for common operations
4. Use `CreateMachineTemplate` with `MachineTemplateConfig` for configuring instances
5. Verify configurations using the flexible validation helper functions like `VerifyMachineSetContainsString`

### Example Test Structure

```go
var _ = Describe("My unified test", framework.LabelDisruptive, Ordered, func() {
    var uf *unified.UnifiedFramework
    var helper *TestHelper

    BeforeAll(func() {
        uf = unified.NewUnifiedFramework()
        cl, _ := framework.LoadClient()
        helper = NewTestHelper(ctx, uf, cl, platform, machineSpec)
    })

    It("creates a MachineSet", func() {
        tpl := helper.CreateTemplate("my-template")
        defer helper.DeleteTemplate(tpl)

        ms := helper.CreateMachineSet("my-ms", tpl, nil)
        defer helper.DeleteMachineSet(ms)

        uf.WaitForMachinesRunning(ctx, cl, ms)
    })
})
```

### Adding New Platform Configurations

To add support for new machine configurations:

1. Define configuration structs in `pkg/unified/config/template_config.go`
2. Implement configuration logic for both MAPI and CAPI backends
3. Add validation in test helpers if needed
