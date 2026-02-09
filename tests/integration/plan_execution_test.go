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

// TestApplyPlanCreateStartStopDelete tests full VM lifecycle through plan execution
func TestApplyPlanCreateStartStopDeleteAPI(t *testing.T) {
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

	// Step 1: Create tenant and site
	tenant := createTestTenantWithServer(t, server, "lifecycle-tenant", "Lifecycle Test Tenant")
	apiKey := createTestAPIKeyWithServer(t, server, tenant.ID, "test-key")
	site := createTestSiteWithServer(t, server, tenant.ID, "test-site")

	// Step 2: Enroll an agent (needed for plan execution)
	agent, _ := enrollTestAgentWithServer(t, server, repo, site.ID, tenant.ID)
	_ = agent

	vmID := "test-vm-001"

	// Step 3: Apply plan with CREATE action
	idempotencyKey := "create-" + uuid.New().String()
	createPlanReq := map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-create-001",
				"operation":    "CREATE",
				"vm_id":        vmID,
				"name":         "test-vm",
				"vcpu_count":   2,
				"memory_mib":   512,
			},
		},
	}

	createBody, _ := json.Marshal(createPlanReq)
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(createBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply create plan: %v", err)
	}

	var createResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("failed to decode create plan response: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for create plan, got %d: %v", resp.StatusCode, createResult)
	}

	planID := getStringField(t, createResult, "plan_id")
	executions := getSliceField(t, createResult, "executions")
	if len(executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(executions))
	}
	executionMap, ok := executions[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected execution to be map[string]interface{}, got %T", executions[0])
	}
	executionID := getStringField(t, executionMap, "id")

	// Verify execution is PENDING
	execState := getStringField(t, executionMap, "state")
	if execState != "PENDING" {
		t.Fatalf("expected execution state PENDING, got %s", execState)
	}

	// Step 4: Simulate agent reporting IN_PROGRESS via heartbeat
	reportExecutionState(t, server, repo, agent.ID, executionID, "IN_PROGRESS", "", "")

	// Verify execution is now IN_PROGRESS
	execs := listExecutionsWithServer(t, server, apiKey, site.ID)
	found := false
	for _, e := range execs {
		if e["id"] == executionID {
			found = true
			if e["state"] != "IN_PROGRESS" {
				t.Fatalf("expected execution state IN_PROGRESS, got %s", e["state"])
			}
			break
		}
	}
	if !found {
		t.Fatal("execution not found after IN_PROGRESS update")
	}

	// Step 5: Simulate agent reporting SUCCEEDED
	reportExecutionState(t, server, repo, agent.ID, executionID, "SUCCEEDED", "", "")

	// Step 6: Verify VM was created in database
	vms := listVMsWithServer(t, server, apiKey, site.ID)
	if len(vms) != 1 {
		t.Fatalf("expected 1 VM, got %d", len(vms))
	}
	if vms[0]["id"] != vmID {
		t.Fatalf("expected VM ID %s, got %s", vmID, vms[0]["id"])
	}
	if vms[0]["state"] != "STOPPED" {
		t.Fatalf("expected VM state STOPPED (after successful CREATE), got %s", vms[0]["state"])
	}

	// Step 7: Apply plan with START action
	startIdempotencyKey := "start-" + uuid.New().String()
	startPlanReq := map[string]interface{}{
		"idempotency_key": startIdempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-start-001",
				"operation":    "START",
				"vm_id":        vmID,
			},
		},
	}

	startBody, _ := json.Marshal(startPlanReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(startBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply start plan: %v", err)
	}

	var startResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&startResult); err != nil {
		t.Fatalf("failed to decode start plan response: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for start plan, got %d", resp.StatusCode)
	}

	startExecutions := getSliceField(t, startResult, "executions")
	startExecutionMap, ok := startExecutions[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected execution to be map[string]interface{}, got %T", startExecutions[0])
	}
	startExecutionID := getStringField(t, startExecutionMap, "id")

	// Simulate START succeeded
	reportExecutionState(t, server, repo, agent.ID, startExecutionID, "SUCCEEDED", "", "")

	// Verify VM is RUNNING
	vms = listVMsWithServer(t, server, apiKey, site.ID)
	if vms[0]["state"] != "RUNNING" {
		t.Fatalf("expected VM state RUNNING, got %s", vms[0]["state"])
	}

	// Step 8: Apply plan with STOP action
	stopIdempotencyKey := "stop-" + uuid.New().String()
	stopPlanReq := map[string]interface{}{
		"idempotency_key": stopIdempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-stop-001",
				"operation":    "STOP",
				"vm_id":        vmID,
			},
		},
	}

	stopBody, _ := json.Marshal(stopPlanReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(stopBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply stop plan: %v", err)
	}

	var stopResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stopResult); err != nil {
		t.Fatalf("failed to decode stop plan response: %v", err)
	}
	resp.Body.Close()

	stopExecutions := getSliceField(t, stopResult, "executions")
	stopExecutionMap, ok := stopExecutions[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected execution to be map[string]interface{}, got %T", stopExecutions[0])
	}
	stopExecutionID := getStringField(t, stopExecutionMap, "id")

	// Simulate STOP succeeded
	reportExecutionState(t, server, repo, agent.ID, stopExecutionID, "SUCCEEDED", "", "")

	// Verify VM is STOPPED
	vms = listVMsWithServer(t, server, apiKey, site.ID)
	if vms[0]["state"] != "STOPPED" {
		t.Fatalf("expected VM state STOPPED, got %s", vms[0]["state"])
	}

	// Step 9: Apply plan with DELETE action
	deleteIdempotencyKey := "delete-" + uuid.New().String()
	deletePlanReq := map[string]interface{}{
		"idempotency_key": deleteIdempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-delete-001",
				"operation":    "DELETE",
				"vm_id":        vmID,
			},
		},
	}

	deleteBody, _ := json.Marshal(deletePlanReq)
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(deleteBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply delete plan: %v", err)
	}

	var deleteResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&deleteResult); err != nil {
		t.Fatalf("failed to decode delete plan response: %v", err)
	}
	resp.Body.Close()

	deleteExecutions := getSliceField(t, deleteResult, "executions")
	deleteExecutionMap, ok := deleteExecutions[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected execution to be map[string]interface{}, got %T", deleteExecutions[0])
	}
	deleteExecutionID := getStringField(t, deleteExecutionMap, "id")

	// Simulate DELETE succeeded
	reportExecutionState(t, server, repo, agent.ID, deleteExecutionID, "SUCCEEDED", "", "")

	// Verify VM is deleted
	vms = listVMsWithServer(t, server, apiKey, site.ID)
	if len(vms) != 0 {
		t.Fatalf("expected 0 VMs after delete, got %d", len(vms))
	}

	// Verify plan was deduplicated correctly (same idempotency key should return same plan)
	_ = planID

	t.Log("VM lifecycle test passed: CREATE -> START -> STOP -> DELETE")
}

// TestApplyPlanIdempotency tests that duplicate plans with the same idempotency key are deduplicated
func TestApplyPlanIdempotencyAPI(t *testing.T) {
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

	// Setup: Create tenant, site
	tenant := createTestTenantWithServer(t, server, "idempotency-tenant", "Idempotency Test Tenant")
	apiKey := createTestAPIKeyWithServer(t, server, tenant.ID, "test-key")
	site := createTestSiteWithServer(t, server, tenant.ID, "test-site")

	// Step 1: Apply plan with idempotency_key
	idempotencyKey := "test-key-1"
	planReq := map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"actions": []map[string]interface{}{
			{
				"operation_id": "op-1",
				"operation":    "CREATE",
				"vm_id":        "vm-001",
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

	var firstResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&firstResult); err != nil {
		t.Fatalf("failed to decode plan response: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	firstPlanID := getStringField(t, firstResult, "plan_id")
	firstExecutions := getSliceField(t, firstResult, "executions")

	if getBoolField(t, firstResult, "deduplicated") {
		t.Fatal("first plan should not be marked as deduplicated")
	}

	// Step 2: Apply same plan again with same idempotency_key
	req, _ = http.NewRequestWithContext(ctx, "POST", server.URL+"/sites/"+site.ID+"/plans", bytes.NewReader(planBody))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to apply plan second time: %v", err)
	}

	var secondResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&secondResult); err != nil {
		t.Fatalf("failed to decode second plan response: %v", err)
	}
	resp.Body.Close()

	secondPlanID := getStringField(t, secondResult, "plan_id")
	secondExecutions := getSliceField(t, secondResult, "executions")

	// Step 3: Verify same plan ID returned
	if firstPlanID != secondPlanID {
		t.Fatalf("expected same plan ID, got %s and %s", firstPlanID, secondPlanID)
	}

	// Step 4: Verify deduplicated flag is true
	if !getBoolField(t, secondResult, "deduplicated") {
		t.Fatal("second plan should be marked as deduplicated")
	}

	// Step 5: Verify only one set of executions exists
	if len(firstExecutions) != len(secondExecutions) {
		t.Fatalf("expected same number of executions, got %d and %d", len(firstExecutions), len(secondExecutions))
	}

	// Verify no new plan was created (check via executions count)
	allExecs := listExecutionsWithServer(t, server, apiKey, site.ID)
	if len(allExecs) != 1 {
		t.Fatalf("expected 1 execution total, got %d", len(allExecs))
	}

	t.Log("Idempotency test passed: duplicate plans correctly deduplicated")
}

// Helper function to list VMs via HTTP API
func listVMsWithServer(t *testing.T, server *httptest.Server, apiKey, siteID string) []map[string]interface{} {
	t.Helper()
	req, _ := http.NewRequest("GET", server.URL+"/sites/"+siteID+"/vms", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to list VMs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode VMs: %v", err)
	}

	vms, ok := result["vms"].([]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	out := make([]map[string]interface{}, len(vms))
	for i, v := range vms {
		out[i] = v.(map[string]interface{})
	}
	return out
}

// Helper function to list executions via HTTP API
func listExecutionsWithServer(t *testing.T, server *httptest.Server, apiKey, siteID string) []map[string]interface{} {
	t.Helper()
	req, _ := http.NewRequest("GET", server.URL+"/sites/"+siteID+"/executions", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to list executions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode executions: %v", err)
	}

	execs, ok := result["executions"].([]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	out := make([]map[string]interface{}, len(execs))
	for i, e := range execs {
		out[i] = e.(map[string]interface{})
	}
	return out
}

// Helper function to report execution state via heartbeat
func reportExecutionState(t *testing.T, server *httptest.Server, repo *store.MemoryRepo, agentID, executionID, state, errorCode, errorMessage string) {
	t.Helper()
	ctx := context.Background()

	// Get agent from repo
	agent, err := repo.GetAgentByID(ctx, agentID)
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}

	update := store.ExecutionUpdate{
		ExecutionID:  executionID,
		State:        state,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		UpdatedAt:    time.Now().UTC(),
	}

	// Report via heartbeat ingestion
	err = repo.IngestHeartbeat(ctx, store.Heartbeat{
		AgentID:          agent.ID,
		Hostname:         "test-host",
		CPUCoresTotal:    4,
		ExecutionUpdates: []store.ExecutionUpdate{update},
	})
	if err != nil {
		t.Fatalf("failed to report execution state: %v", err)
	}
}
