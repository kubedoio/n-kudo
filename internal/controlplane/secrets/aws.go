package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// AWSSecretsManagerStore implements SecretStore using AWS Secrets Manager.
// This is a simplified implementation that uses environment-based configuration.
// For production use, you should use the AWS SDK directly.
type AWSSecretsManagerStore struct {
	mu       sync.RWMutex
	region   string
	prefix   string
	// In-memory cache for fallback when AWS SDK is not available
	cache    map[string]string
	// Track if we're in mock mode (no AWS SDK)
	mockMode bool
}

// awsSecretResponse represents the expected response structure
type awsSecretResponse struct {
	ARN           string            `json:"ARN,omitempty"`
	Name          string            `json:"Name,omitempty"`
	VersionID     string            `json:"VersionId,omitempty"`
	SecretString  string            `json:"SecretString,omitempty"`
	SecretBinary  []byte            `json:"SecretBinary,omitempty"`
	VersionStages []string          `json:"VersionStages,omitempty"`
	CreatedDate   time.Time         `json:"CreatedDate,omitempty"`
}

// NewAWSSecretsManagerStore creates a new AWS Secrets Manager store
func NewAWSSecretsManagerStore(cfg Config) (*AWSSecretsManagerStore, error) {
	region := cfg.AWSRegion
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	store := &AWSSecretsManagerStore{
		region:   region,
		prefix:   "nkudo/",
		cache:    make(map[string]string),
		mockMode: true, // Default to mock mode unless AWS SDK is available
	}

	// Check if AWS credentials are available
	if hasAWSCredentials() {
		store.mockMode = false
	}

	return store, nil
}

// hasAWSCredentials checks if AWS credentials are available
func hasAWSCredentials() bool {
	// Check for AWS credentials in environment
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}
	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" {
		return true
	}
	// Check for IAM role (running on EC2/EKS)
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" {
		return true
	}
	return false
}

// normalizeKey normalizes the key for AWS Secrets Manager
// AWS secret names can contain alphanumeric characters and the characters /_+=.@-
func (a *AWSSecretsManagerStore) normalizeKey(key string) string {
	key = strings.Trim(key, "/")
	if a.prefix != "" {
		return a.prefix + key
	}
	return key
}

// Get retrieves a secret from AWS Secrets Manager
func (a *AWSSecretsManagerStore) Get(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	normalizedKey := a.normalizeKey(key)

	// In mock mode, use cache
	if a.mockMode {
		if value, ok := a.cache[normalizedKey]; ok {
			return value, nil
		}
		// Try to read from environment as fallback
		envKey := "AWS_SECRET_" + strings.ToUpper(strings.ReplaceAll(key, "/", "_"))
		if value := os.Getenv(envKey); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("%w: %s (AWS Secrets Manager mock mode)", ErrSecretNotFound, key)
	}

	// In real mode, we would use the AWS SDK
	// For now, return not implemented to indicate SDK is needed
	return "", fmt.Errorf("%w: AWS SDK integration required for %s", ErrNotImplemented, key)
}

// Set stores a secret in AWS Secrets Manager
func (a *AWSSecretsManagerStore) Set(key string, value string) error {
	if key == "" {
		return ErrInvalidKey
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	normalizedKey := a.normalizeKey(key)

	// In mock mode, store in cache
	if a.mockMode {
		a.cache[normalizedKey] = value
		return nil
	}

	// In real mode, we would use the AWS SDK
	return fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// Delete removes a secret from AWS Secrets Manager
func (a *AWSSecretsManagerStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	normalizedKey := a.normalizeKey(key)

	// In mock mode, remove from cache
	if a.mockMode {
		delete(a.cache, normalizedKey)
		return nil
	}

	// In real mode, we would use the AWS SDK
	return fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// GetSecret retrieves a secret with additional metadata
// This method is specific to AWS and not part of the SecretStore interface
func (a *AWSSecretsManagerStore) GetSecret(key string) (*awsSecretResponse, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	normalizedKey := a.normalizeKey(key)

	if a.mockMode {
		if value, ok := a.cache[normalizedKey]; ok {
			return &awsSecretResponse{
				Name:         normalizedKey,
				SecretString: value,
				CreatedDate:  time.Now(),
			}, nil
		}
		return nil, ErrSecretNotFound
	}

	return nil, fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// CreateSecret creates a new secret (fails if secret already exists)
func (a *AWSSecretsManagerStore) CreateSecret(key, value string, tags map[string]string) error {
	if key == "" {
		return ErrInvalidKey
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	normalizedKey := a.normalizeKey(key)

	if a.mockMode {
		if _, exists := a.cache[normalizedKey]; exists {
			return fmt.Errorf("secret %s already exists", key)
		}
		a.cache[normalizedKey] = value
		return nil
	}

	return fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// UpdateSecret updates an existing secret (creates new version)
func (a *AWSSecretsManagerStore) UpdateSecret(key, value string) error {
	// Same as Set for AWS
	return a.Set(key, value)
}

// RotateSecret initiates rotation of a secret
func (a *AWSSecretsManagerStore) RotateSecret(key, lambdaARN string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mockMode {
		return fmt.Errorf("%w: rotation not available in mock mode", ErrNotImplemented)
	}

	return fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// ListSecrets lists all secrets with the given prefix
func (a *AWSSecretsManagerStore) ListSecrets(prefix string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.mockMode {
		var keys []string
		for k := range a.cache {
			if strings.HasPrefix(k, a.prefix+prefix) {
				// Remove prefix from key
				key := strings.TrimPrefix(k, a.prefix)
				keys = append(keys, key)
			}
		}
		return keys, nil
	}

	return nil, fmt.Errorf("%w: AWS SDK integration required", ErrNotImplemented)
}

// GetRegion returns the AWS region
func (a *AWSSecretsManagerStore) GetRegion() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.region
}

// IsMockMode returns true if running in mock mode (no AWS SDK)
func (a *AWSSecretsManagerStore) IsMockMode() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.mockMode
}

// EnableMockMode forces mock mode (useful for testing)
func (a *AWSSecretsManagerStore) EnableMockMode() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mockMode = true
}

// ParseSecretString parses a JSON secret string into a map
func ParseSecretString(secretString string) (map[string]string, error) {
	var result map[string]string
	if err := json.Unmarshal([]byte(secretString), &result); err != nil {
		return nil, fmt.Errorf("parse secret string: %w", err)
	}
	return result, nil
}

// BuildSecretString builds a JSON secret string from a map
func BuildSecretString(data map[string]string) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("build secret string: %w", err)
	}
	return string(bytes), nil
}

// Ensure AWSSecretsManagerStore implements SecretStore
var _ SecretStore = (*AWSSecretsManagerStore)(nil)
