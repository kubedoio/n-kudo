package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// HashiCorpVaultStore implements SecretStore using HashiCorp Vault's KV v2 API
type HashiCorpVaultStore struct {
	mu        sync.RWMutex
	client    *http.Client
	baseURL   string
	token     string
	mountPath string // Usually "secret" for KV v2
	basePath  string // e.g., "nkudo" or "data/nkudo"
}

// vaultResponse represents the Vault API response structure
type vaultResponse struct {
	Data struct {
		Data map[string]interface{} `json:"data"`
		Metadata struct {
			Version      int       `json:"version"`
			CreatedTime  time.Time `json:"created_time"`
			DeletionTime string    `json:"deletion_time"`
			Destroyed    bool      `json:"destroyed"`
		} `json:"metadata,omitempty"`
	} `json:"data"`
	Errors []string `json:"errors,omitempty"`
}

// vaultListResponse represents the Vault list API response
type vaultListResponse struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
	Errors []string `json:"errors,omitempty"`
}

// NewHashiCorpVaultStore creates a new Vault secret store
func NewHashiCorpVaultStore(cfg Config) (*HashiCorpVaultStore, error) {
	if cfg.VaultAddr == "" {
		return nil, fmt.Errorf("VAULT_ADDR is required for Vault store")
	}
	if cfg.VaultToken == "" {
		return nil, fmt.Errorf("VAULT_TOKEN is required for Vault store")
	}

	baseURL := strings.TrimSuffix(cfg.VaultAddr, "/")

	// Parse and normalize the path
	mountPath := "secret" // Default mount path for KV v2
	basePath := cfg.VaultPath
	if basePath == "" {
		basePath = "nkudo"
	}

	return &HashiCorpVaultStore{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		token:     cfg.VaultToken,
		mountPath: mountPath,
		basePath:  strings.Trim(basePath, "/"),
	}, nil
}

// buildURL constructs the full URL for a Vault API request
func (v *HashiCorpVaultStore) buildURL(key string) string {
	// KV v2 API path: /v1/{mountPath}/data/{path}/{key}
	key = strings.Trim(key, "/")
	fullPath := path.Join(v.basePath, key)
	return fmt.Sprintf("%s/v1/%s/data/%s", v.baseURL, v.mountPath, fullPath)
}

// buildMetadataURL constructs the URL for metadata operations
func (v *HashiCorpVaultStore) buildMetadataURL(key string) string {
	key = strings.Trim(key, "/")
	fullPath := path.Join(v.basePath, key)
	return fmt.Sprintf("%s/v1/%s/metadata/%s", v.baseURL, v.mountPath, fullPath)
}

// buildListURL constructs the URL for list operations
func (v *HashiCorpVaultStore) buildListURL(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	fullPath := path.Join(v.basePath, prefix)
	return fmt.Sprintf("%s/v1/%s/metadata/%s?list=true", v.baseURL, v.mountPath, fullPath)
}

// doRequest performs an HTTP request to Vault
func (v *HashiCorpVaultStore) doRequest(method, urlStr string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return v.client.Do(req)
}

// Get retrieves a secret from Vault
func (v *HashiCorpVaultStore) Get(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	url := v.buildURL(key)
	resp, err := v.doRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrSecretNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	var result vaultResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	// The actual secret data is nested under data.data
	if data, ok := result.Data.Data["value"]; ok {
		if str, ok := data.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", data), nil
	}

	// If no "value" field, return the first field or serialized data
	if len(result.Data.Data) > 0 {
		for _, v := range result.Data.Data {
			if str, ok := v.(string); ok {
				return str, nil
			}
			return fmt.Sprintf("%v", v), nil
		}
	}

	return "", ErrSecretNotFound
}

// Set stores a secret in Vault
func (v *HashiCorpVaultStore) Set(key string, value string) error {
	if key == "" {
		return ErrInvalidKey
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	url := v.buildURL(key)

	payload := map[string]interface{}{
		"data": map[string]string{
			"value": value,
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := v.doRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Vault returns 200 for update, 204 for create
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete removes a secret from Vault (soft delete - creates a delete version)
func (v *HashiCorpVaultStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	url := v.buildURL(key)

	resp, err := v.doRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	// Vault returns 204 on successful delete
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Destroy permanently removes a secret version from Vault
func (v *HashiCorpVaultStore) Destroy(key string, versions []int) error {
	if key == "" {
		return ErrInvalidKey
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	url := v.buildMetadataURL(key)

	payload := map[string]interface{}{
		"versions": versions,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Use the destroy endpoint
	destroyURL := strings.Replace(url, "/metadata/", "/destroy/", 1)
	resp, err := v.doRequest("POST", destroyURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// List retrieves all secret keys under a given prefix
func (v *HashiCorpVaultStore) List(prefix string) ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	url := v.buildListURL(prefix)
	resp, err := v.doRequest("LIST", url, nil)
	if err != nil {
		// Try GET with list=true query param if LIST method not supported
		resp, err = v.doRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("vault request failed: %w", err)
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	var result vaultListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data.Keys, nil
}

// Health checks if Vault is reachable and the token is valid
func (v *HashiCorpVaultStore) Health() error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	url := fmt.Sprintf("%s/v1/sys/health", v.baseURL)
	resp, err := v.doRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Vault returns 200 for initialized, unsealed, and active
	// 429 if unsealed and standby
	// 472 if disaster recovery mode replication secondary
	// 473 if performance standby
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 429 && resp.StatusCode != 473 {
		return fmt.Errorf("vault unhealthy (status %d)", resp.StatusCode)
	}

	return nil
}

// GetData retrieves all fields of a secret as a map
func (v *HashiCorpVaultStore) GetData(key string) (map[string]string, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	url := v.buildURL(key)
	resp, err := v.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSecretNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	var result vaultResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	data := make(map[string]string)
	for k, v := range result.Data.Data {
		if str, ok := v.(string); ok {
			data[k] = str
		} else {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	return data, nil
}

// SetData stores multiple fields in a secret
func (v *HashiCorpVaultStore) SetData(key string, data map[string]string) error {
	if key == "" {
		return ErrInvalidKey
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	url := v.buildURL(key)

	// Convert map[string]string to map[string]interface{}
	dataInterface := make(map[string]interface{}, len(data))
	for k, v := range data {
		dataInterface[k] = v
	}

	payload := map[string]interface{}{
		"data": dataInterface,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := v.doRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Ensure HashiCorpVaultStore implements SecretStore
var _ SecretStore = (*HashiCorpVaultStore)(nil)
