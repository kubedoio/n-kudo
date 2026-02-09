# Phase 1 Task 3: Backend Executions List Endpoint

## Task Description
Implement the backend API endpoint for listing plan executions with filtering.

## Acceptance Criteria
- [ ] `GET /sites/{siteId}/executions` endpoint returns list of executions
- [ ] Supports `status` query param filter (comma-separated: PENDING,IN_PROGRESS,SUCCEEDED,FAILED)
- [ ] Supports `limit` query param for pagination
- [ ] Proper tenant authorization via API key

## API Specification

### GET /sites/{siteId}/executions?status=SUCCEEDED,FAILED&limit=50
**Auth:** X-API-Key

Response 200:
```json
{
  "executions": [
    {
      "id": "uuid",
      "plan_id": "uuid",
      "operation_id": "string",
      "operation_type": "CREATE",
      "state": "SUCCEEDED",
      "vm_id": "uuid",
      "error_code": null,
      "error_message": null,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:30Z"
    }
  ]
}
```

## Database
Table: `executions`

Query needs to:
1. Join with plans table to filter by site_id
2. Join with sites table to verify tenant access
3. Support status filtering
4. Support limit

## Files to Modify
- `internal/controlplane/db/store.go` - Add interface method
- `internal/controlplane/db/postgres.go` - Implement query
- `internal/controlplane/api/server.go` - Add HTTP handler
- `internal/controlplane/db/memory.go` - Implement for in-memory store

## New Type
```go
type ExecutionWithTimestamps struct {
    ID            string    `json:"id"`
    PlanID        string    `json:"plan_id"`
    OperationID   string    `json:"operation_id"`
    OperationType string    `json:"operation_type"`
    State         string    `json:"state"`
    VMID          string    `json:"vm_id"`
    ErrorCode     *string   `json:"error_code,omitempty"`
    ErrorMessage  *string   `json:"error_message,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

## Testing
- Unit tests for DB query with filters
- Integration tests for HTTP endpoint
- Test tenant isolation

## Definition of Done
- [ ] Code implemented
- [ ] Tests passing
- [ ] Manual verification with curl
- [ ] Filters work correctly

## Estimated Effort
4-5 hours
