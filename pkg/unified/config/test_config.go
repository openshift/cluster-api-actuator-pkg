package config

import (
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
func LoadTestConfig() *TestConfig {
	config := &TestConfig{
		BackendType:      BackendTypeMAPI, // Default to MAPI
		AuthoritativeAPI: BackendTypeMAPI, // Default to MAPI
	}

	// Read configuration from environment variables
	if backendType := os.Getenv("TEST_BACKEND_TYPE"); backendType != "" {
		switch strings.ToUpper(backendType) {
		case "MAPI":
			config.BackendType = BackendTypeMAPI
		case "CAPI":
			config.BackendType = BackendTypeCAPI
		}
	}

	if authAPI := os.Getenv("TEST_AUTHORITATIVE_API"); authAPI != "" {
		switch strings.ToUpper(authAPI) {
		case "MAPI":
			config.AuthoritativeAPI = BackendTypeMAPI
		case "CAPI":
			config.AuthoritativeAPI = BackendTypeCAPI
		}
	}

	return config
}
