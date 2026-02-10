package secrets

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// mockSecretStore is a simple in-memory implementation for testing
type mockSecretStore struct {
	data map[string]string
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{
		data: make(map[string]string),
	}
}

func (m *mockSecretStore) Get(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}
	if value, ok := m.data[key]; ok {
		return value, nil
	}
	return "", ErrSecretNotFound
}

func (m *mockSecretStore) Set(key string, value string) error {
	if key == "" {
		return ErrInvalidKey
	}
	m.data[key] = value
	return nil
}

func (m *mockSecretStore) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	delete(m.data, key)
	return nil
}

func TestNewStore_Env(t *testing.T) {
	cfg := Config{Type: StoreTypeEnv}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	_, ok := store.(*EnvSecretStore)
	if !ok {
		t.Error("Expected EnvSecretStore")
	}
}

func TestNewStore_Default(t *testing.T) {
	cfg := Config{Type: ""}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	_, ok := store.(*EnvSecretStore)
	if !ok {
		t.Error("Expected EnvSecretStore as default")
	}
}

func TestNewStore_InvalidType(t *testing.T) {
	cfg := Config{Type: "invalid"}
	_, err := NewStore(cfg)
	if err == nil {
		t.Error("Expected error for invalid store type")
	}
}

func TestNewStore_Custom(t *testing.T) {
	mock := newMockSecretStore()
	cfg := Config{CustomStore: mock}
	store, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store != mock {
		t.Error("Expected custom store")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Save and restore environment
	oldType := os.Getenv("SECRET_STORE_TYPE")
	oldVaultAddr := os.Getenv("VAULT_ADDR")
	defer func() {
		os.Setenv("SECRET_STORE_TYPE", oldType)
		os.Setenv("VAULT_ADDR", oldVaultAddr)
	}()

	os.Setenv("SECRET_STORE_TYPE", "vault")
	os.Setenv("VAULT_ADDR", "https://vault.example.com")

	cfg := LoadConfigFromEnv()
	if cfg.Type != StoreTypeVault {
		t.Errorf("Expected type vault, got %s", cfg.Type)
	}
	if cfg.VaultAddr != "https://vault.example.com" {
		t.Errorf("Expected vault addr, got %s", cfg.VaultAddr)
	}
}

func TestLoadConfigFromEnv_Default(t *testing.T) {
	// Clear environment
	os.Unsetenv("SECRET_STORE_TYPE")

	cfg := LoadConfigFromEnv()
	if cfg.Type != StoreTypeEnv {
		t.Errorf("Expected default type env, got %s", cfg.Type)
	}
}

func TestGetWithFallback(t *testing.T) {
	mock := newMockSecretStore()
	mock.Set("test-key", "from-store")

	// Should get from store
	value := GetWithFallback(mock, "test-key", "TEST_ENV", "default")
	if value != "from-store" {
		t.Errorf("Expected 'from-store', got %s", value)
	}

	// Should fall back to env
	os.Setenv("TEST_ENV_FALLBACK", "from-env")
	defer os.Unsetenv("TEST_ENV_FALLBACK")
	value = GetWithFallback(mock, "nonexistent", "TEST_ENV_FALLBACK", "default")
	if value != "from-env" {
		t.Errorf("Expected 'from-env', got %s", value)
	}

	// Should fall back to default
	value = GetWithFallback(mock, "nonexistent", "NONEXISTENT_ENV", "default-value")
	if value != "default-value" {
		t.Errorf("Expected 'default-value', got %s", value)
	}
}

func TestMustGet(t *testing.T) {
	mock := newMockSecretStore()
	mock.Set("existing", "value")

	// Should get existing
	value := MustGet(mock, "existing")
	if value != "value" {
		t.Errorf("Expected 'value', got %s", value)
	}

	// Should panic for non-existing
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for missing key")
		}
	}()
	MustGet(mock, "nonexistent")
}

// EnvSecretStore Tests

func TestEnvSecretStore_Get(t *testing.T) {
	store := NewEnvSecretStore()

	// Set env var
	os.Setenv("NKUDO_TEST_SECRET", "secret-value")
	defer os.Unsetenv("NKUDO_TEST_SECRET")

	value, err := store.Get("test.secret")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("Expected 'secret-value', got %s", value)
	}
}

func TestEnvSecretStore_Get_NotFound(t *testing.T) {
	store := NewEnvSecretStore()

	_, err := store.Get("nonexistent.key.12345")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Errorf("Expected ErrSecretNotFound, got %v", err)
	}
}

func TestEnvSecretStore_Get_InvalidKey(t *testing.T) {
	store := NewEnvSecretStore()

	_, err := store.Get("")
	if !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}
}

func TestEnvSecretStore_Set(t *testing.T) {
	store := NewEnvSecretStore()

	err := store.Set("my-key", "my-value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := store.Get("my-key")
	if err != nil {
		t.Fatalf("Get after Set failed: %v", err)
	}
	if value != "my-value" {
		t.Errorf("Expected 'my-value', got %s", value)
	}
}

func TestEnvSecretStore_Delete(t *testing.T) {
	store := NewEnvSecretStore()

	store.Set("delete-me", "value")
	err := store.Delete("delete-me")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("delete-me")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Error("Expected key to be deleted")
	}
}

func TestEnvSecretStore_toEnvKey(t *testing.T) {
	store := NewEnvSecretStore()

	tests := []struct {
		input    string
		expected string
	}{
		{"test.key", "NKUDO_TEST_KEY"},
		{"test/key", "NKUDO_TEST_KEY"},
		{"test-key", "NKUDO_TEST_KEY"},
		{"TEST.KEY", "NKUDO_TEST_KEY"},
	}

	for _, tt := range tests {
		result := store.toEnvKey(tt.input)
		if result != tt.expected {
			t.Errorf("toEnvKey(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestEnvSecretStore_WithPrefix(t *testing.T) {
	store := NewEnvSecretStoreWithPrefix("CUSTOM_")

	os.Setenv("CUSTOM_SECRET", "prefixed-value")
	defer os.Unsetenv("CUSTOM_SECRET")

	value, err := store.Get("secret")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "prefixed-value" {
		t.Errorf("Expected 'prefixed-value', got %s", value)
	}
}

func TestEnvSecretStore_GetDatabaseURL(t *testing.T) {
	store := NewEnvSecretStore()

	os.Setenv("DATABASE_URL", "postgres://test")
	defer os.Unsetenv("DATABASE_URL")

	url := store.GetDatabaseURL()
	if url != "postgres://test" {
		t.Errorf("Expected database URL, got %s", url)
	}
}

func TestEnvSecretStore_GetAdminKey(t *testing.T) {
	store := NewEnvSecretStore()

	os.Setenv("ADMIN_KEY", "admin-secret")
	defer os.Unsetenv("ADMIN_KEY")

	key := store.GetAdminKey()
	if key != "admin-secret" {
		t.Errorf("Expected admin key, got %s", key)
	}
}

func TestEnvSecretStore_GetSMTPPassword(t *testing.T) {
	store := NewEnvSecretStore()

	os.Setenv("SMTP_PASSWORD", "smtp-secret")
	defer os.Unsetenv("SMTP_PASSWORD")

	pass := store.GetSMTPPassword()
	if pass != "smtp-secret" {
		t.Errorf("Expected SMTP password, got %s", pass)
	}
}

// Vault Store Tests (mock-based)

func TestHashiCorpVaultStore_ValidateConfig(t *testing.T) {
	// Missing address
	_, err := NewHashiCorpVaultStore(Config{VaultToken: "token"})
	if err == nil {
		t.Error("Expected error for missing VAULT_ADDR")
	}

	// Missing token
	_, err = NewHashiCorpVaultStore(Config{VaultAddr: "http://vault:8200"})
	if err == nil {
		t.Error("Expected error for missing VAULT_TOKEN")
	}
}

func TestHashiCorpVaultStore_buildURL(t *testing.T) {
	store, _ := NewHashiCorpVaultStore(Config{
		VaultAddr:  "http://vault:8200",
		VaultToken: "token",
		VaultPath:  "nkudo",
	})

	url := store.buildURL("db/password")
	expected := "http://vault:8200/v1/secret/data/nkudo/db/password"
	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

func TestHashiCorpVaultStore_buildURL_NoPath(t *testing.T) {
	store, _ := NewHashiCorpVaultStore(Config{
		VaultAddr:  "http://vault:8200/",
		VaultToken: "token",
	})

	url := store.buildURL("secret")
	expected := "http://vault:8200/v1/secret/data/nkudo/secret"
	if url != expected {
		t.Errorf("Expected URL %s, got %s", expected, url)
	}
}

// AWS Store Tests

func TestNewAWSSecretsManagerStore_DefaultRegion(t *testing.T) {
	// Clear AWS region env
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_DEFAULT_REGION")

	store, err := NewAWSSecretsManagerStore(Config{})
	if err != nil {
		t.Fatalf("NewAWSSecretsManagerStore failed: %v", err)
	}

	if store.GetRegion() != "us-east-1" {
		t.Errorf("Expected default region us-east-1, got %s", store.GetRegion())
	}
}

func TestNewAWSSecretsManagerStore_FromEnv(t *testing.T) {
	os.Setenv("AWS_REGION", "eu-west-1")
	defer os.Unsetenv("AWS_REGION")

	store, err := NewAWSSecretsManagerStore(Config{})
	if err != nil {
		t.Fatalf("NewAWSSecretsManagerStore failed: %v", err)
	}

	if store.GetRegion() != "eu-west-1" {
		t.Errorf("Expected region eu-west-1, got %s", store.GetRegion())
	}
}

func TestNewAWSSecretsManagerStore_FromConfig(t *testing.T) {
	os.Unsetenv("AWS_REGION")

	store, err := NewAWSSecretsManagerStore(Config{AWSRegion: "ap-southeast-1"})
	if err != nil {
		t.Fatalf("NewAWSSecretsManagerStore failed: %v", err)
	}

	if store.GetRegion() != "ap-southeast-1" {
		t.Errorf("Expected region ap-southeast-1, got %s", store.GetRegion())
	}
}

func TestAWSSecretsManagerStore_Get_MockMode(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})
	store.EnableMockMode()

	// Should return not found for unknown key
	_, err := store.Get("unknown")
	if !errors.Is(err, ErrSecretNotFound) {
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected not found error, got %v", err)
		}
	}
}

func TestAWSSecretsManagerStore_Set_MockMode(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})
	store.EnableMockMode()

	err := store.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := store.Get("test-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got %s", value)
	}
}

func TestAWSSecretsManagerStore_Delete_MockMode(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})
	store.EnableMockMode()

	store.Set("delete-me", "value")
	err := store.Delete("delete-me")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get("delete-me")
	if err == nil {
		t.Error("Expected error for deleted key")
	}
}

func TestAWSSecretsManagerStore_CreateSecret(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})
	store.EnableMockMode()

	err := store.CreateSecret("new-secret", "value", nil)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Should fail if already exists
	err = store.CreateSecret("new-secret", "other", nil)
	if err == nil {
		t.Error("Expected error for duplicate secret")
	}
}

func TestAWSSecretsManagerStore_ListSecrets(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})
	store.EnableMockMode()

	store.Set("prefix/secret1", "value1")
	store.Set("prefix/secret2", "value2")
	store.Set("other/secret3", "value3")

	keys, err := store.ListSecrets("prefix/")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestAWSSecretsManagerStore_IsMockMode(t *testing.T) {
	store, _ := NewAWSSecretsManagerStore(Config{})

	// By default should be in mock mode (no AWS credentials)
	if !store.IsMockMode() {
		t.Error("Expected mock mode by default when no credentials")
	}
}

// Utility function tests

func TestParseSecretString(t *testing.T) {
	jsonStr := `{"username":"admin","password":"secret123"}`

	result, err := ParseSecretString(jsonStr)
	if err != nil {
		t.Fatalf("ParseSecretString failed: %v", err)
	}

	if result["username"] != "admin" {
		t.Errorf("Expected username 'admin', got %s", result["username"])
	}
	if result["password"] != "secret123" {
		t.Errorf("Expected password 'secret123', got %s", result["password"])
	}
}

func TestParseSecretString_Invalid(t *testing.T) {
	_, err := ParseSecretString("not valid json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestBuildSecretString(t *testing.T) {
	data := map[string]string{
		"username": "admin",
		"password": "secret123",
	}

	result, err := BuildSecretString(data)
	if err != nil {
		t.Fatalf("BuildSecretString failed: %v", err)
	}

	expected := `{"password":"secret123","username":"admin"}`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// Integration-style tests

func TestStore_RoundTrip(t *testing.T) {
	stores := []struct {
		name  string
		store SecretStore
	}{
		{"Env", NewEnvSecretStore()},
	}

	for _, tc := range stores {
		t.Run(tc.name, func(t *testing.T) {
			// Set
			err := tc.store.Set("key1", "value1")
			if err != nil {
				t.Fatalf("Set failed: %v", err)
			}

			// Get
			value, err := tc.store.Get("key1")
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if value != "value1" {
				t.Errorf("Expected 'value1', got %s", value)
			}

			// Update
			err = tc.store.Set("key1", "value2")
			if err != nil {
				t.Fatalf("Set (update) failed: %v", err)
			}

			value, _ = tc.store.Get("key1")
			if value != "value2" {
				t.Errorf("Expected 'value2', got %s", value)
			}

			// Delete
			err = tc.store.Delete("key1")
			if err != nil {
				t.Fatalf("Delete failed: %v", err)
			}

			_, err = tc.store.Get("key1")
			if err == nil {
				t.Error("Expected error for deleted key")
			}
		})
	}
}

func TestEnvSecretStore_ConcurrentAccess(t *testing.T) {
	store := NewEnvSecretStore()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := fmt.Sprintf("key%d", n)
			value := fmt.Sprintf("value%d", n)
			store.Set(key, value)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all writes
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		expected := fmt.Sprintf("value%d", i)
		value, err := store.Get(key)
		if err != nil {
			t.Errorf("Failed to get %s: %v", key, err)
		}
		if value != expected {
			t.Errorf("Expected %s, got %s", expected, value)
		}
	}
}
