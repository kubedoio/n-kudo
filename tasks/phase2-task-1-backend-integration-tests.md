# Phase 2 Task 1: Backend Integration Tests

## Task Description
Implement comprehensive integration tests for the control plane backend.

## Prerequisites
- Phase 1 complete (all endpoints implemented)
- Existing test infrastructure in `tests/integration/`

## Acceptance Criteria
- [ ] `TestApplyPlanCreateStartStopDelete` - Full VM lifecycle test
- [ ] `TestApplyPlanIdempotency` - Idempotency verification
- [ ] `TestCrossTenantIsolationRejected` - Security boundary test
- [ ] `TestExecutionLogStreaming` - Log ingestion/retrieval test
- [ ] `TestAgentEnrollmentFlow` - End-to-end enrollment test
- [ ] All tests pass with `go test ./tests/integration/...`

## Test Implementation Details

### Test 1: TestApplyPlanCreateStartStopDelete
**File:** `tests/integration/plan_execution_test.go`

```go
func TestApplyPlanCreateStartStopDelete(t *testing.T) {
    // Setup: Create tenant, site, enroll agent
    // Step 1: Apply plan with CREATE action
    // Step 2: Verify execution created with PENDING status
    // Step 3: Simulate agent reporting IN_PROGRESS
    // Step 4: Simulate agent reporting SUCCEEDED
    // Step 5: Verify VM created in database
    // Step 6: Repeat for START, STOP, DELETE
    // Step 7: Verify VM state changes correctly
}
```

### Test 2: TestApplyPlanIdempotency
**File:** `tests/integration/plan_execution_test.go`

```go
func TestApplyPlanIdempotency(t *testing.T) {
    // Setup: Create tenant, site
    // Step 1: Apply plan with idempotency_key "test-key-1"
    // Step 2: Get plan ID from response
    // Step 3: Apply same plan with same idempotency_key
    // Step 4: Verify same plan ID returned (deduplicated: true)
    // Step 5: Verify only one set of executions exists
}
```

### Test 3: TestCrossTenantIsolationRejected
**File:** `tests/integration/security_test.go`

```go
func TestCrossTenantIsolationRejected(t *testing.T) {
    // Setup: Create tenant A and tenant B
    // Step 1: Create API key for tenant A
    // Step 2: Try to access tenant B's sites with tenant A's key
    // Step 3: Verify 403 Forbidden
    // Step 4: Try to apply plan to tenant B's site
    // Step 5: Verify 403 Forbidden
    // Step 6: Verify tenant A can access own resources
}
```

### Test 4: TestExecutionLogStreaming
**File:** `tests/integration/logs_test.go`

```go
func TestExecutionLogStreaming(t *testing.T) {
    // Setup: Create tenant, site, execution
    // Step 1: Ingest logs via /agents/logs endpoint
    // Step 2: Query logs via /executions/{id}/logs
    // Step 3: Verify logs match
    // Step 4: Test limit parameter
    // Step 5: Test ordering (by sequence)
}
```

### Test 5: TestAgentEnrollmentFlow
**File:** `tests/integration/enrollment_test.go`

```go
func TestAgentEnrollmentFlow(t *testing.T) {
    // Setup: Create tenant, site, issue enrollment token
    // Step 1: Call /enroll with token, hostname, CSR
    // Step 2: Verify agent created
    // Step 3: Verify host created
    // Step 4: Verify client certificate returned
    // Step 5: Send heartbeat with new certificate
    // Step 6: Verify host facts updated
    // Step 7: Try to reuse enrollment token
    // Step 8: Verify token rejected (one-time use)
}
```

## Test Infrastructure

Use the existing test setup:
- `tests/integration/` package
- PostgreSQL test container (or in-memory store)
- HTTP test server

## Files to Create/Modify

### New Files
- `tests/integration/plan_execution_test.go`
- `tests/integration/security_test.go`
- `tests/integration/logs_test.go`
- `tests/integration/enrollment_test.go`

### Modified Files
- May need to add test helpers to `tests/integration/mocks/`

## Test Data Helpers

Create helper functions:
```go
func createTestTenant(t *testing.T, repo store.Repo) *store.Tenant
func createTestSite(t *testing.T, repo store.Repo, tenantID string) *store.Site
func createTestAPIKey(t *testing.T, repo store.Repo, tenantID string) string
func enrollTestAgent(t *testing.T, repo store.Repo, siteID string) *store.Agent
```

## Running Tests

```bash
cd /srv/data01/kubedo/n-kudo
go test ./tests/integration/... -v
```

## Definition of Done
- [ ] All 5 integration tests implemented
- [ ] Tests pass consistently
- [ ] Tests clean up after themselves
- [ ] No race conditions
- [ ] go vet clean

## Estimated Effort
6-8 hours
