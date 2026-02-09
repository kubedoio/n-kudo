# Phase 1 Task 2: Backend Token History Endpoint

## Task Description
Implement the backend API endpoint for listing enrollment token history.

## Acceptance Criteria
- [ ] `GET /tenants/{tenantId}/enrollment-tokens` endpoint returns list of enrollment tokens
- [ ] Includes both used and unused tokens
- [ ] Shows consumption status and timestamp if used
- [ ] Proper authorization (tenant-scoped)

## API Specification

### GET /tenants/{tenantId}/enrollment-tokens
**Auth:** X-Admin-Key or X-API-Key

Response 200:
```json
{
  "tokens": [
    {
      "id": "uuid",
      "site_id": "uuid",
      "site_name": "string",
      "created_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-01-01T00:15:00Z",
      "consumed": true,
      "consumed_at": "2024-01-01T00:05:00Z",
      "consumed_by_agent_id": "uuid"
    }
  ]
}
```

## Database
Tables: `enrollment_tokens`, `sites` (for site_name join)

Query needs to:
1. Filter by tenant_id
2. Join with sites to get site_name
3. Left join with agents to get consumption info

## Files to Modify
- `internal/controlplane/db/store.go` - Add interface method
- `internal/controlplane/db/postgres.go` - Implement query
- `internal/controlplane/api/server.go` - Add HTTP handler
- `internal/controlplane/db/memory.go` - Implement for in-memory store

## New Type
```go
type EnrollmentTokenWithStatus struct {
    ID                string     `json:"id"`
    SiteID            string     `json:"site_id"`
    SiteName          string     `json:"site_name"`
    CreatedAt         time.Time  `json:"created_at"`
    ExpiresAt         time.Time  `json:"expires_at"`
    Consumed          bool       `json:"consumed"`
    ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
    ConsumedByAgentID *string    `json:"consumed_by_agent_id,omitempty"`
}
```

## Testing
- Unit tests for DB query
- Integration tests for HTTP endpoint
- Test that token hash is NEVER returned

## Definition of Done
- [ ] Code implemented
- [ ] Tests passing
- [ ] Manual verification with curl
- [ ] Response excludes sensitive fields (token_hash)

## Estimated Effort
3-4 hours
