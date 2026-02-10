package secrets

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// EnvSecretStore implements SecretStore using environment variables.
// Keys are prefixed with "NKUDO_" and uppercased for the environment.
// This is the default/fallback implementation.
type EnvSecretStore struct {
	mu     sync.RWMutex
	prefix string
	cache  map[string]string // In-memory cache for values set via Set()
}

// NewEnvSecretStore creates a new environment variable secret store.
// It uses NKUDO_ prefix for all keys.
func NewEnvSecretStore() *EnvSecretStore {
	return &EnvSecretStore{
		prefix: "NKUDO_",
		cache:  make(map[string]string),
	}
}

// NewEnvSecretStoreWithPrefix creates a new environment variable secret store
// with a custom prefix.
func NewEnvSecretStoreWithPrefix(prefix string) *EnvSecretStore {
	return &EnvSecretStore{
		prefix: prefix,
		cache:  make(map[string]string),
	}
}

// toEnvKey converts a secret key to an environment variable name
func (e *EnvSecretStore) toEnvKey(key string) string {
	// Replace dots and slashes with underscores, uppercase
	normalized := strings.ToUpper(key)
	normalized = strings.ReplaceAll(normalized, ".", "_")
	normalized = strings.ReplaceAll(normalized, "/", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return e.prefix + normalized
}

// Get retrieves a secret from environment variables
func (e *EnvSecretStore) Get(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// First check the in-memory cache (for values set via Set())
	if value, ok := e.cache[key]; ok {
		return value, nil
	}

	// Then check environment variables
	envKey := e.toEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		// Also try without prefix for common secrets
		value = os.Getenv(strings.ToUpper(key))
		if value == "" {
			return "", fmt.Errorf("%w: %s (env: %s)", ErrSecretNotFound, key, envKey)
		}
	}

	return value, nil
}

// Set stores a secret in the in-memory cache (cannot set env vars permanently)
func (e *EnvSecretStore) Set(key string, value string) error {
	if key == "" {
		return ErrInvalidKey
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.cache[key] = value
	return nil
}

// Delete removes a secret from the in-memory cache
func (e *EnvSecretStore) Delete(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.cache, key)
	// Cannot delete environment variables, but we can remove from cache
	return nil
}

// List returns all secret keys known to the store (from cache and env)
// Note: This only returns keys that have been accessed or set
func (e *EnvSecretStore) List() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := make([]string, 0, len(e.cache))
	for k := range e.cache {
		keys = append(keys, k)
	}

	// Also scan environment for NKUDO_ prefixed vars
	prefixLen := len(e.prefix)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, e.prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				// Convert back from env format to key format
				key := strings.ToLower(parts[0][prefixLen:])
				keys = append(keys, key)
			}
		}
	}

	return keys
}

// GetDatabaseURL retrieves the database URL from environment
// Checks multiple common environment variable names
func (e *EnvSecretStore) GetDatabaseURL() string {
	candidates := []string{
		"DATABASE_URL",
		"NKUDO_DATABASE_URL",
		"POSTGRES_URL",
		"DB_URL",
	}

	for _, env := range candidates {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}

	return ""
}

// GetAdminKey retrieves the admin key from environment
func (e *EnvSecretStore) GetAdminKey() string {
	candidates := []string{
		"ADMIN_KEY",
		"NKUDO_ADMIN_KEY",
		"API_ADMIN_KEY",
	}

	for _, env := range candidates {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}

	return ""
}

// GetSMTPPassword retrieves the SMTP password from environment
func (e *EnvSecretStore) GetSMTPPassword() string {
	candidates := []string{
		"SMTP_PASSWORD",
		"NKUDO_SMTP_PASSWORD",
		"EMAIL_PASSWORD",
	}

	for _, env := range candidates {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}

	return ""
}
