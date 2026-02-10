package grpc

import (
	"context"
	"testing"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// mockCA implements CAInterface for testing
type mockCA struct{}

func (m *mockCA) CertPEM() []byte { return []byte("mock-cert") }
func (m *mockCA) CertPool() *interface{} { return nil }
func (m *mockCA) SignAgentCSR(csrPEM []byte, agentID, tenantID, siteID string, ttl time.Duration) (certPEM []byte, serial string, err error) {
	return []byte("mock-cert"), "mock-serial", nil
}

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Enabled {
		t.Error("Expected default Enabled to be false")
	}
	
	if cfg.ListenAddr != ":50051" {
		t.Errorf("Expected default ListenAddr to be :50051, got %s", cfg.ListenAddr)
	}
}

func TestServerCreation(t *testing.T) {
	// Server can be created with nil CA (TLS won't be configured)
	cfg := Config{
		Enabled:    false,
		ListenAddr: ":0",
	}
	
	server := NewServer(cfg, nil, nil, 15*time.Second, 45*time.Second, 2, 24*time.Hour)
	
	if server == nil {
		t.Fatal("Expected server to be created")
	}
	
	if server.IsStarted() {
		t.Error("Expected server to not be started initially")
	}
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := LoggingInterceptor()
	if interceptor == nil {
		t.Error("Expected interceptor to be created")
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := RecoveryInterceptor()
	if interceptor == nil {
		t.Error("Expected interceptor to be created")
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxTenantIDKey, "test-tenant")
	
	tenantID, ok := GetTenantIDFromContext(ctx)
	if !ok {
		t.Error("Expected to get tenant ID from context")
	}
	
	if tenantID != "test-tenant" {
		t.Errorf("Expected tenant ID to be test-tenant, got %s", tenantID)
	}
}

func TestGetTenantIDFromContext_NotFound(t *testing.T) {
	ctx := context.Background()
	
	_, ok := GetTenantIDFromContext(ctx)
	if ok {
		t.Error("Expected to not find tenant ID in empty context")
	}
}

func TestGetAgentFromContext(t *testing.T) {
	agent := store.Agent{
		ID:       "agent-123",
		TenantID: "tenant-456",
		SiteID:   "site-789",
	}
	
	ctx := context.WithValue(context.Background(), ctxAgentIDKey, agent)
	
	retrievedAgent, ok := GetAgentFromContext(ctx)
	if !ok {
		t.Error("Expected to get agent from context")
	}
	
	if retrievedAgent.ID != agent.ID {
		t.Errorf("Expected agent ID to be %s, got %s", agent.ID, retrievedAgent.ID)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test firstNonEmpty
	result := firstNonEmpty("", "b", "c")
	if result != "b" {
		t.Errorf("Expected firstNonEmpty to return 'b', got %s", result)
	}
	
	result = firstNonEmpty("", "", "")
	if result != "" {
		t.Errorf("Expected firstNonEmpty to return '', got %s", result)
	}
	
	// Test valueOr
	result = valueOr("", "fallback")
	if result != "fallback" {
		t.Errorf("Expected valueOr to return 'fallback', got %s", result)
	}
	
	result = valueOr("value", "fallback")
	if result != "value" {
		t.Errorf("Expected valueOr to return 'value', got %s", result)
	}
	
	// Test hashString
	hash := hashString("test")
	if hash == "" {
		t.Error("Expected hashString to return non-empty string")
	}
	
	// Hash should be deterministic
	hash2 := hashString("test")
	if hash != hash2 {
		t.Error("Expected hashString to be deterministic")
	}
	
	// Different inputs should produce different outputs
	hash3 := hashString("different")
	if hash == hash3 {
		t.Error("Expected different inputs to produce different hashes")
	}
}

func TestValidateRequiredString(t *testing.T) {
	// Test empty string
	err := validateRequiredString("", "field")
	if err == nil {
		t.Error("Expected error for empty string")
	}
	
	// Test whitespace-only string
	err = validateRequiredString("   ", "field")
	if err == nil {
		t.Error("Expected error for whitespace-only string")
	}
	
	// Test valid string
	err = validateRequiredString("value", "field")
	if err != nil {
		t.Errorf("Expected no error for valid string, got: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Enabled {
		t.Error("Default config should have Enabled=false")
	}
	
	if cfg.ListenAddr != ":50051" {
		t.Errorf("Default ListenAddr should be :50051, got %s", cfg.ListenAddr)
	}
}
