package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	controlplane "github.com/kubedoio/n-kudo/internal/controlplane/api"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
)

// TestExecutionLogStreaming tests log ingestion and retrieval
func TestExecutionLogStreamingAPI(t *testing.T) {
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

	// Step 1: Create tenant, site, and enroll agent
	tenant := createTestTenantWithServer(t, server, "logs-tenant", "Logs Test Tenant")
	apiKey := createTestAPIKeyWithServer(t, server, tenant.ID, "test-key")
	site := createTestSiteWithServer(t, server, tenant.ID, "test-site")

	// Create a plan and execution
	idempotencyKey := "logs-" + uuid.New().String()
	planReq := map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-logs-001",
				"operation":    "CREATE",
				"vm_id":        "vm-logs-001",
				"name":         "test-vm",
			},
		},
	}

	planBody, _ := json.Marshal(planReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(planBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply plan: %v", err)
	}

	var planResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&planResult); err != nil {
		t.Fatalf("failed to decode plan response: %v", err)
	}
	resp.Body.Close()

	executions := planResult["executions"].([]interface{})
	executionID := executions[0].(map[string]interface{})["id"].(string)

	// Step 2: Ingest logs via the repository directly (simulating agent log ingestion)
	agent, _ := enrollTestAgentWithServer(t, server, repo, site.ID, tenant.ID)

	logEntries := []store.LogIngestEntry{
		{
			ExecutionID: executionID,
			Sequence:    1,
			Severity:    "INFO",
			Message:     "Starting VM creation",
			EmittedAt:   time.Now().UTC().Add(-3 * time.Second),
		},
		{
			ExecutionID: executionID,
			Sequence:    2,
			Severity:    "INFO",
			Message:     "Allocating resources",
			EmittedAt:   time.Now().UTC().Add(-2 * time.Second),
		},
		{
			ExecutionID: executionID,
			Sequence:    3,
			Severity:    "INFO",
			Message:     "VM created successfully",
			EmittedAt:   time.Now().UTC().Add(-1 * time.Second),
		},
	}

	accepted, dropped, err := repo.IngestLogs(ctx, store.LogIngest{
		AgentID: agent.ID,
		Entries: logEntries,
	})
	if err != nil {
		t.Fatalf("failed to ingest logs: %v", err)
	}
	if dropped > 0 {
		t.Fatalf("expected 0 dropped logs, got %d", dropped)
	}
	if accepted != int64(len(logEntries)) {
		t.Fatalf("expected %d accepted logs, got %d", len(logEntries), accepted)
	}

	// Step 3: Query logs via /executions/{id}/logs
	logsURL := server.URL + "/executions/" + executionID + "/logs"
	req, _ = http.NewRequestWithContext(ctx, "GET", logsURL, nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	var logsResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&logsResult); err != nil {
		t.Fatalf("failed to decode logs response: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	logs, ok := logsResult["logs"].([]interface{})
	if !ok {
		t.Fatal("logs field not found in response")
	}
	if len(logs) != len(logEntries) {
		t.Fatalf("expected %d logs, got %d", len(logEntries), len(logs))
	}

	// Step 4: Verify logs match
	for i, logEntry := range logs {
		entry := logEntry.(map[string]interface{})
		expected := logEntries[i]

		if entry["message"] != expected.Message {
			t.Fatalf("log %d: expected message %q, got %q", i, expected.Message, entry["message"])
		}
		if entry["severity"] != expected.Severity {
			t.Fatalf("log %d: expected severity %q, got %q", i, expected.Severity, entry["severity"])
		}
		if entry["execution_id"] != expected.ExecutionID {
			t.Fatalf("log %d: expected execution_id %q, got %q", i, expected.ExecutionID, entry["execution_id"])
		}
	}

	// Step 5: Test limit parameter
	req, _ = http.NewRequestWithContext(ctx, "GET", logsURL+"?limit=2", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get logs with limit: %v", err)
	}

	var limitedResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&limitedResult); err != nil {
		t.Fatalf("failed to decode limited logs response: %v", err)
	}
	resp.Body.Close()

	limitedLogs, ok := limitedResult["logs"].([]interface{})
	if !ok {
		t.Fatal("logs field not found in limited response")
	}
	if len(limitedLogs) != 2 {
		t.Fatalf("expected 2 logs with limit, got %d", len(limitedLogs))
	}

	// Step 6: Test ordering (by sequence)
	for i := 0; i < len(logs)-1; i++ {
		curr := logs[i].(map[string]interface{})["sequence"].(float64)
		next := logs[i+1].(map[string]interface{})["sequence"].(float64)
		if curr >= next {
			t.Fatalf("logs not sorted by sequence: %v >= %v", curr, next)
		}
	}

	t.Log("Execution log streaming test passed")
}

// TestLogIngestionUnauthorized validates unauthorized log access is rejected
func TestLogIngestionUnauthorized(t *testing.T) {
	ctx := context.Background()
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

	// Create tenant A and B
	tenantA := createTestTenantWithServer(t, server, "logs-tenant-a", "Logs Test Tenant A")
	tenantB := createTestTenantWithServer(t, server, "logs-tenant-b", "Logs Test Tenant B")
	apiKeyA := createTestAPIKeyWithServer(t, server, tenantA.ID, "key-a")
	apiKeyB := createTestAPIKeyWithServer(t, server, tenantB.ID, "key-b")
	siteA := createTestSiteWithServer(t, server, tenantA.ID, "site-a")

	// Create a plan and execution for tenant A
	idempotencyKey := "logs-auth-" + uuid.New().String()
	planReq := map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-auth-001",
				"operation":    "CREATE",
				"vm_id":        "vm-auth-001",
			},
		},
	}

	planBody, _ := json.Marshal(planReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+siteA.ID+"/plans", bytes.NewReader(planBody))
	req.Header.Set("X-API-Key", apiKeyA)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply plan: %v", err)
	}

	var planResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&planResult); err != nil {
		t.Fatalf("failed to decode plan response: %v", err)
	}
	resp.Body.Close()

	executions := planResult["executions"].([]interface{})
	executionID := executions[0].(map[string]interface{})["id"].(string)

	// Try to access execution logs from tenant B with tenant A's execution ID
	logsURL := server.URL + "/executions/" + executionID + "/logs"
	req, _ = http.NewRequestWithContext(ctx, "GET", logsURL, nil)
	req.Header.Set("X-API-Key", apiKeyB)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	resp.Body.Close()

	// Should get 404 (not found) since execution doesn't belong to tenant B
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 404 or 403 for cross-tenant log access, got %d", resp.StatusCode)
	}

	// Verify tenant A can still access their own logs
	req, _ = http.NewRequestWithContext(ctx, "GET", logsURL, nil)
	req.Header.Set("X-API-Key", apiKeyA)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for own tenant logs, got %d", resp.StatusCode)
	}

	t.Log("Log authorization test passed")
}
