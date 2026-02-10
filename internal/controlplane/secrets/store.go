package secrets

import (
	"errors"
	"fmt"
	"os"
)

// SecretStore defines the interface for secret storage backends.
// Implementations must be safe for concurrent use.
type SecretStore interface {
	// Get retrieves a secret value by key
	Get(key string) (string, error)
	// Set stores a secret value by key
	Set(key string, value string) error
	// Delete removes a secret by key
	Delete(key string) error
}

// Common errors
var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrInvalidKey     = errors.New("invalid secret key")
	ErrNotImplemented = errors.New("operation not implemented")
)

// StoreType represents the type of secret store
type StoreType string

const (
	// StoreTypeEnv uses environment variables for secrets
	StoreTypeEnv StoreType = "env"
	// StoreTypeVault uses HashiCorp Vault for secrets
	StoreTypeVault StoreType = "vault"
	// StoreTypeAWS uses AWS Secrets Manager for secrets
	StoreTypeAWS StoreType = "aws"
)

// Config holds configuration for creating a secret store
type Config struct {
	Type StoreType
	// Vault-specific configuration
	VaultAddr  string
	VaultToken string
	VaultPath  string // Base path for secrets (e.g., "secret/nkudo")
	// AWS-specific configuration
	AWSRegion string
	// For testing - allow injecting a custom store
	CustomStore SecretStore
}

// LoadConfigFromEnv creates a Config from environment variables
func LoadConfigFromEnv() Config {
	storeType := StoreType(os.Getenv("SECRET_STORE_TYPE"))
	if storeType == "" {
		storeType = StoreTypeEnv // Default to env
	}

	return Config{
		Type:       storeType,
		VaultAddr:  os.Getenv("VAULT_ADDR"),
		VaultToken: os.Getenv("VAULT_TOKEN"),
		VaultPath:  os.Getenv("VAULT_PATH"),
		AWSRegion:  os.Getenv("AWS_REGION"),
	}
}

// NewStore creates a SecretStore based on the provided configuration.
// Returns an error if the configuration is invalid or the store cannot be created.
func NewStore(cfg Config) (SecretStore, error) {
	// Allow custom store injection for testing
	if cfg.CustomStore != nil {
		return cfg.CustomStore, nil
	}

	switch cfg.Type {
	case StoreTypeEnv, "":
		return NewEnvSecretStore(), nil
	case StoreTypeVault:
		return NewHashiCorpVaultStore(cfg)
	case StoreTypeAWS:
		return NewAWSSecretsManagerStore(cfg)
	default:
		return nil, fmt.Errorf("unsupported secret store type: %s", cfg.Type)
	}
}

// NewStoreFromEnv creates a SecretStore using configuration from environment variables
func NewStoreFromEnv() (SecretStore, error) {
	cfg := LoadConfigFromEnv()
	return NewStore(cfg)
}

// Helper function to get a secret with fallback to environment variable
func GetWithFallback(store SecretStore, key, envKey, defaultValue string) string {
	// Try the secret store first
	if store != nil {
		value, err := store.Get(key)
		if err == nil && value != "" {
			return value
		}
	}

	// Fall back to environment variable
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}

	return defaultValue
}

// MustGet retrieves a secret from the store or panics if not found.
// Useful for required secrets like database passwords.
func MustGet(store SecretStore, key string) string {
	value, err := store.Get(key)
	if err != nil {
		panic(fmt.Sprintf("required secret %q not found: %v", key, err))
	}
	return value
}
