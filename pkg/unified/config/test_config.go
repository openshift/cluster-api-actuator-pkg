package config

import (
	"fmt"
	"os"
	"strings"
)

// BackendType represents the type of backend.
type BackendType string

const (
	// BackendTypeMAPI represents MAPI backend.
	BackendTypeMAPI BackendType = "MAPI"
	// BackendTypeCAPI represents CAPI backend.
	BackendTypeCAPI BackendType = "CAPI"
)

// TestConfig defines test configuration.
type TestConfig struct {
	// Backend type: MAPI or CAPI
	BackendType BackendType

	// Authoritative API type: MAPI or CAPI
	AuthoritativeAPI BackendType
}

// LoadTestConfig loads test configuration from environment variables.
// Returns error if environment variables contain invalid values.
func LoadTestConfig() (*TestConfig, error) {
	config := &TestConfig{
		BackendType:      BackendTypeMAPI, // Default to MAPI
		AuthoritativeAPI: BackendTypeMAPI, // Default to MAPI
	}

	// Read and validate TEST_BACKEND_TYPE
	if backendType := strings.TrimSpace(os.Getenv("TEST_BACKEND_TYPE")); backendType != "" {
		backendTypeUpper := strings.ToUpper(backendType)
		switch backendTypeUpper {
		case "MAPI":
			config.BackendType = BackendTypeMAPI
		case "CAPI":
			config.BackendType = BackendTypeCAPI
		default:
			return nil, fmt.Errorf("invalid TEST_BACKEND_TYPE value %q: must be 'MAPI' or 'CAPI'", backendType)
		}
	}

	// Read and validate TEST_AUTHORITATIVE_API
	if authAPI := strings.TrimSpace(os.Getenv("TEST_AUTHORITATIVE_API")); authAPI != "" {
		authAPIUpper := strings.ToUpper(authAPI)
		switch authAPIUpper {
		case "MAPI":
			config.AuthoritativeAPI = BackendTypeMAPI
		case "CAPI":
			config.AuthoritativeAPI = BackendTypeCAPI
		default:
			return nil, fmt.Errorf("invalid TEST_AUTHORITATIVE_API value %q: must be 'MAPI' or 'CAPI'", authAPI)
		}
	}

	return config, nil
}

// GetTestConfigOrDie loads test config and panics on error.
func GetTestConfigOrDie() *TestConfig {
	cfg, err := LoadTestConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load test config: %v", err))
	}

	return cfg
}
