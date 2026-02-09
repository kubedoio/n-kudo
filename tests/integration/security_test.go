package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	controlplane "github.com/kubedoio/n-kudo/internal/controlplane/api"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// TestCrossTenantIsolationRejected validates that Tenant A cannot access Tenant B's resources
func TestCrossTenantIsolationRejectedAPI(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryRepo()

	// Create control plane app
	cfg := controlplane.Config{
		AdminKey:              "test-admin-key",
		CACommonName:          "test-ca",
		RequirePersistentPKI:  false,
		DefaultTokenTTL:       3600,
		HeartbeatInterval:     15,
		MaxPlansPerHeartbeat:  10,
		PlanLeaseTTL:          30,
		OfflineSweepInterval:  0,
		OfflineAfter:          60,
	}

	app, err := controlplane.NewApp(cfg, repo)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	// Create Tenant A
	tenantA := createTestTenantWithServer(t, server, "tenant-a", "Tenant A")

	// Create Tenant B
	tenantB := createTestTenantWithServer(t, server, "tenant-b", "Tenant B")

	// Create API key for Tenant A
	apiKeyA := createTestAPIKeyWithServer(t, server, tenantA.ID, "key-a")

	// Create API key for Tenant B
	apiKeyB := createTestAPIKeyWithServer(t, server, tenantB.ID, "key-b")

	// Create site for Tenant A
	siteA := createTestSiteWithServer(t, server, tenantA.ID, "site-a")

	// Create site for Tenant B
	siteB := createTestSiteWithServer(t, server, tenantB.ID, "site-b")

	// Test 1: Try to access Tenant B's sites with Tenant A's key (should fail)
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/tenants/"+tenantB.ID+"/sites", nil)
	req.Header.Set("X-API-Key", apiKeyA)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when accessing other tenant, got %d", resp.StatusCode)
	}

	// Test 2: Try to apply plan to Tenant B's site using Tenant A's key (should fail)
	planReq := map[string]interface{}{
		"idempotency_key": "test-key-1",
		"actions": []map[string]interface{}{
			{"operation_id": "op-1", "operation": "CREATE", "vm_id": "vm-1", "name": "test-vm"},
		},
	}
	planBody, _ := json.Marshal(planReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+siteB.ID+"/plans", bytes.NewReader(planBody))
	req.Header.Set("X-API-Key", apiKeyA)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	// This should fail because siteB doesn't belong to tenantA
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 when applying plan to other tenant's site, got %d", resp.StatusCode)
	}

	// Test 3: Try to access Tenant A's sites using Tenant B's key (should fail)
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/tenants/"+tenantA.ID+"/sites", nil)
	req.Header.Set("X-API-Key", apiKeyB)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when accessing other tenant's sites, got %d", resp.StatusCode)
	}

	// Test 4: Verify Tenant A can access own resources
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/tenants/"+tenantA.ID+"/sites", nil)
	req.Header.Set("X-API-Key", apiKeyA)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK when accessing own sites, got %d", resp.StatusCode)
	}

	// Test 5: Verify Tenant B can access own resources
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/tenants/"+tenantB.ID+"/sites", nil)
	req.Header.Set("X-API-Key", apiKeyB)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK when accessing own sites, got %d", resp.StatusCode)
	}

	// Test 6: Try to list hosts from Tenant A's site using Tenant B's key
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/sites/"+siteA.ID+"/hosts", nil)
	req.Header.Set("X-API-Key", apiKeyB)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 when accessing other tenant's hosts, got %d", resp.StatusCode)
	}

	// Test 7: Try to list VMs from Tenant A's site using Tenant B's key
	req, _ = http.NewRequestWithContext(ctx, "GET", server.URL+"/sites/"+siteA.ID+"/vms", nil)
	req.Header.Set("X-API-Key", apiKeyB)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 when accessing other tenant's VMs, got %d", resp.StatusCode)
	}

	// Verify correct site IDs
	_ = siteA
	_ = siteB

	t.Log("Cross-tenant isolation test passed: all unauthorized accesses were rejected")
}

// TestInvalidAPIKeyRejection validates that invalid API keys are rejected
func TestInvalidAPIKeyRejection(t *testing.T) {
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

	// Create a tenant
	tenant := createTestTenantWithServer(t, server, "test-tenant", "Test Tenant")

	// Test with invalid API key
	req, _ := http.NewRequest("GET", server.URL+"/tenants/"+tenant.ID+"/sites", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid key, got %d", resp.StatusCode)
	}

	// Test with missing API key
	req, _ = http.NewRequest("GET", server.URL+"/tenants/"+tenant.ID+"/sites", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing key, got %d", resp.StatusCode)
	}

	t.Log("Invalid API key rejection test passed")
}

// Helper function to create a test tenant via HTTP API
func createTestTenantWithServer(t *testing.T, server *httptest.Server, slug, name string) *store.Tenant {
	t.Helper()
	reqBody := map[string]interface{}{
		"slug":                slug,
		"name":                name,
		"primary_region":      "local",
		"data_retention_days": 30,
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", server.URL+"/tenants", bytes.NewReader(body))
	req.Header.Set("X-Admin-Key", "test-admin-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var tenant store.Tenant
	if err := json.NewDecoder(resp.Body).Decode(&tenant); err != nil {
		t.Fatalf("failed to decode tenant: %v", err)
	}
	return &tenant
}

// Helper function to create a test API key via HTTP API
func createTestAPIKeyWithServer(t *testing.T, server *httptest.Server, tenantID, name string) string {
	t.Helper()
	reqBody := map[string]interface{}{
		"name": name,
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", server.URL+"/tenants/"+tenantID+"/api-keys", bytes.NewReader(body))
	req.Header.Set("X-Admin-Key", "test-admin-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode API key: %v", err)
	}
	return result["api_key"].(string)
}

// Helper function to create a test site via HTTP API
func createTestSiteWithServer(t *testing.T, server *httptest.Server, tenantID, name string) *store.Site {
	t.Helper()

	// First create an API key
	apiKey := createTestAPIKeyWithServer(t, server, tenantID, "test-key")

	reqBody := map[string]interface{}{
		"name": name,
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", server.URL+"/tenants/"+tenantID+"/sites", bytes.NewReader(body))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create site: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
	}

	var site store.Site
	if err := json.NewDecoder(resp.Body).Decode(&site); err != nil {
		t.Fatalf("failed to decode site: %v", err)
	}
	return &site
}

// Helper function to generate a unique ID
func generateID() string {
	return uuid.New().String()
}
