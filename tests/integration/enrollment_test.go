package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	controlplane "github.com/kubedoio/n-kudo/internal/controlplane/api"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// TestAgentEnrollmentFlow tests the full enrollment flow
func TestAgentEnrollmentFlow(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryRepo()

	cfg := controlplane.Config{
		AdminKey:             "test-admin-key",
		CACommonName:         "test-ca",
		RequirePersistentPKI: false,
		DefaultTokenTTL:      3600,
		HeartbeatInterval:    15,
		MaxPlansPerHeartbeat: 10,
		PlanLeaseTTL:         30,
	}

	app, err := controlplane.NewApp(cfg, repo)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	// Step 1: Create tenant, site, and issue enrollment token
	tenant := createTestTenantWithServer(t, server, "enroll-tenant", "Enrollment Test Tenant")
	apiKey := createTestAPIKeyWithServer(t, server, tenant.ID, "test-key")
	site := createTestSiteWithServer(t, server, tenant.ID, "test-site")

	// Issue enrollment token
	tokenReq := map[string]interface{}{
		"site_id":            site.ID,
		"expires_in_seconds": 3600,
	}
	tokenBody, _ := json.Marshal(tokenReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/tenants/"+tenant.ID+"/enrollment-tokens", bytes.NewReader(tokenBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to issue enrollment token: %v", err)
	}

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	resp.Body.Close()

	plainToken := getStringField(t, tokenResult, "token")

	// Step 2: Call /enroll with token, hostname, and CSR
	csrPEM := generateTestCSR(t)
	enrollReq := map[string]interface{}{
		"enrollment_token": plainToken,
		"agent_version":    "test-1.0.0",
		"requested_hostname": "test-agent-host",
		"csr_pem":          csrPEM,
		"bootstrap_nonce":  "test-nonce-123",
	}

	enrollBody, _ := json.Marshal(enrollReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/enroll", bytes.NewReader(enrollBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to enroll: %v", err)
	}

	var enrollResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&enrollResult); err != nil {
		t.Fatalf("failed to decode enroll response: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for enrollment, got %d: %v", resp.StatusCode, enrollResult)
	}

	// Step 3: Verify agent created
	agentID := getStringField(t, enrollResult, "agent_id")
	if agentID == "" {
		t.Fatal("agent_id not returned in enrollment response")
	}

	// Verify agent exists in repository
	agent, err := repo.GetAgentByID(ctx, agentID)
	if err != nil {
		t.Fatalf("failed to get agent from repo: %v", err)
	}
	if agent.ID != agentID {
		t.Fatalf("agent ID mismatch: expected %s, got %s", agentID, agent.ID)
	}
	if agent.TenantID != tenant.ID {
		t.Fatalf("agent tenant ID mismatch: expected %s, got %s", tenant.ID, agent.TenantID)
	}
	if agent.SiteID != site.ID {
		t.Fatalf("agent site ID mismatch: expected %s, got %s", site.ID, agent.SiteID)
	}

	// Step 4: Verify host created
	hosts := listHostsWithServer(t, server, apiKey, site.ID)
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0]["hostname"] != "test-agent-host" {
		t.Fatalf("expected hostname 'test-agent-host', got %s", hosts[0]["hostname"])
	}

	// Step 5: Verify client certificate returned
	clientCertPEM := getStringField(t, enrollResult, "client_certificate_pem")
	if clientCertPEM == "" {
		t.Fatal("client_certificate_pem not returned")
	}

	caCertPEM := enrollResult["ca_certificate_pem"].(string)
	if caCertPEM == "" {
		t.Fatal("ca_certificate_pem not returned")
	}

	refreshToken := getStringField(t, enrollResult, "refresh_token")
	if refreshToken == "" {
		t.Fatal("refresh_token not returned")
	}

	// Step 6: Send heartbeat with agent context (simulated)
	// Note: In a real scenario, this would use mTLS. Here we test via repo directly
	err = repo.IngestHeartbeat(ctx, store.Heartbeat{
		AgentID:                  agentID,
		Hostname:                 "test-agent-host",
		AgentVersion:             "test-1.0.0",
		OS:                       "linux",
		Arch:                     "amd64",
		CPUCoresTotal:            8,
		MemoryBytesTotal:         16384,
		StorageBytesTotal:        1024000,
		KVMAvailable:             true,
		CloudHypervisorAvailable: true,
	})
	if err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	// Step 7: Verify host facts updated
	hosts = listHostsWithServer(t, server, apiKey, site.ID)
	if hosts[0]["cpu_cores_total"] != float64(8) {
		t.Fatalf("expected cpu_cores_total 8, got %v", hosts[0]["cpu_cores_total"])
	}
	if hosts[0]["memory_bytes_total"] != float64(16384) {
		t.Fatalf("expected memory_bytes_total 16384, got %v", hosts[0]["memory_bytes_total"])
	}
	if hosts[0]["kvm_available"] != true {
		t.Fatalf("expected kvm_available true, got %v", hosts[0]["kvm_available"])
	}

	// Step 8: Try to reuse token (should fail)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/enroll", bytes.NewReader(enrollBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send second enrollment request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for token reuse, got %d", resp.StatusCode)
	}

	t.Log("Agent enrollment flow test passed")
}

// TestExpiredEnrollmentToken validates expired tokens are rejected
func TestExpiredEnrollmentToken(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryRepo()

	cfg := controlplane.Config{
		AdminKey:             "test-admin-key",
		CACommonName:         "test-ca",
		RequirePersistentPKI: false,
		DefaultTokenTTL:      1, // Very short TTL
	}

	app, err := controlplane.NewApp(cfg, repo)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	// Create tenant and site
	tenant := createTestTenantWithServer(t, server, "expired-tenant", "Expired Token Test Tenant")
	apiKey := createTestAPIKeyWithServer(t, server, tenant.ID, "test-key")
	site := createTestSiteWithServer(t, server, tenant.ID, "test-site")

	// Issue enrollment token with very short expiry
	tokenReq := map[string]interface{}{
		"site_id":            site.ID,
		"expires_in_seconds": 1, // 1 second
	}
	tokenBody, _ := json.Marshal(tokenReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/tenants/"+tenant.ID+"/enrollment-tokens", bytes.NewReader(tokenBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to issue enrollment token: %v", err)
	}

	var tokenResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResult); err != nil {
		t.Fatalf("failed to decode token response: %v", err)
	}
	resp.Body.Close()

	plainToken := getStringField(t, tokenResult, "token")

	// Wait for token to expire (poll for up to 5 seconds)
	start := time.Now()
	expired := false
	for time.Since(start) < 5*time.Second {
		time.Sleep(100 * time.Millisecond)
		// Try a test request to see if token is expired
		csrPEMTest := generateTestCSR(t)
		testReq := map[string]interface{}{
			"enrollment_token":   plainToken,
			"agent_version":      "test-1.0.0",
			"requested_hostname": "test-agent-host",
			"csr_pem":            csrPEMTest,
			"bootstrap_nonce":    "test-nonce-123",
		}
		testBody, _ := json.Marshal(testReq)
		reqTest, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/enroll", bytes.NewReader(testBody))
		reqTest.Header.Set("Content-Type", "application/json")
		respTest, err := http.DefaultClient.Do(reqTest)
		if err != nil {
			t.Fatalf("failed to send test enrollment request: %v", err)
		}
		respTest.Body.Close()
		if respTest.StatusCode == http.StatusUnauthorized {
			expired = true
			break
		}
	}
	if !expired {
		t.Fatal("token did not expire within expected time")
	}

	// Try to enroll with expired token
	csrPEM := generateTestCSR(t)
	enrollReq := map[string]interface{}{
		"enrollment_token":   plainToken,
		"agent_version":      "test-1.0.0",
		"requested_hostname": "test-agent-host",
		"csr_pem":            csrPEM,
		"bootstrap_nonce":    "test-nonce-123",
	}

	enrollBody, _ := json.Marshal(enrollReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/enroll", bytes.NewReader(enrollBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send enrollment request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", resp.StatusCode)
	}

	t.Log("Expired token test passed")
}

// TestInvalidEnrollmentToken validates invalid tokens are rejected
func TestInvalidEnrollmentToken(t *testing.T) {
	repo := store.NewMemoryRepo()

	cfg := controlplane.Config{
		AdminKey:             "test-admin-key",
		CACommonName:         "test-ca",
		RequirePersistentPKI: false,
	}

	app, err := controlplane.NewApp(cfg, repo)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	// Try to enroll with invalid token
	csrPEM := generateTestCSR(t)
	enrollReq := map[string]interface{}{
		"enrollment_token":   "invalid-token",
		"agent_version":      "test-1.0.0",
		"requested_hostname": "test-agent-host",
		"csr_pem":            csrPEM,
		"bootstrap_nonce":    "test-nonce-123",
	}

	enrollBody, _ := json.Marshal(enrollReq)
	req, _ := http.NewRequest("POST", server.URL+"/v1/enroll", bytes.NewReader(enrollBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to send enrollment request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d", resp.StatusCode)
	}

	t.Log("Invalid token test passed")
}

// Helper function to list hosts via HTTP API
func listHostsWithServer(t *testing.T, server *httptest.Server, apiKey, siteID string) []map[string]interface{} {
	t.Helper()
	req, _ := http.NewRequest("GET", server.URL+"/sites/"+siteID+"/hosts", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to list hosts: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode hosts: %v", err)
	}

	hosts, ok := result["hosts"].([]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	out := make([]map[string]interface{}, len(hosts))
	for i, h := range hosts {
		out[i] = h.(map[string]interface{})
	}
	return out
}

// Helper function to generate a test CSR
func generateTestCSR(t *testing.T) string {
	t.Helper()
	
	// Generate a proper RSA key and CSR for testing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: "test-agent",
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		t.Fatalf("failed to create CSR: %v", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	return string(csrPEM)
}

// Helper function to enroll a test agent directly via repository
func enrollTestAgentWithServer(t *testing.T, server *httptest.Server, repo *store.MemoryRepo, siteID, tenantID string) (*store.Agent, string) {
	t.Helper()
	ctx := context.Background()

	// Create enrollment token directly in repo
	plainToken := "test-token-" + uuid.New().String()
	tokenHash := hashStringForTest(plainToken)

	_, err := repo.IssueEnrollmentToken(ctx, store.EnrollmentToken{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		SiteID:    siteID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to issue enrollment token: %v", err)
	}

	// Consume token and create agent
	consumeResult, err := repo.ConsumeEnrollmentToken(ctx, tokenHash, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to consume enrollment token: %v", err)
	}

	agent := store.Agent{
		ID:               uuid.New().String(),
		TenantID:         consumeResult.TenantID,
		SiteID:           consumeResult.SiteID,
		HostID:           uuid.New().String(),
		CertSerial:       "test-serial",
		RefreshTokenHash: "test-refresh-hash",
		AgentVersion:     "test-1.0.0",
		OS:               "linux",
		Arch:             "amd64",
		KernelVersion:    "5.0.0",
		State:            "ONLINE",
	}

	createdAgent, err := repo.CreateAgentFromEnrollment(ctx, consumeResult.TokenID, agent, "test-host")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	return &createdAgent, plainToken
}

// Simple hash function for testing
func hashStringForTest(s string) string {
	// Simple hash - in real code this uses sha256
	return "hash:" + s
}
