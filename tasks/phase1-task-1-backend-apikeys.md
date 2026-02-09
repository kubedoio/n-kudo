# Phase 1 Task 1: Backend API Key Management Endpoints

## Task Description
Implement the missing backend API endpoints for listing and revoking API keys.

## Acceptance Criteria
- [ ] `GET /tenants/{tenantId}/api-keys` endpoint returns list of API keys for tenant
- [ ] `DELETE /tenants/{tenantId}/api-keys/{keyId}` endpoint revokes (deletes) an API key
- [ ] API keys should NOT return the actual key (only metadata)
- [ ] Proper authorization checks (X-Admin-Key or X-API-Key)
- [ ] Returns 404 if tenant or key not found
- [ ] Returns 403 if key doesn't belong to tenant

## API Specification

### GET /tenants/{tenantId}/api-keys
**Auth:** X-Admin-Key or X-API-Key

Response 200:
```json
{
  "api_keys": [
    {
      "id": "uuid",
      "tenant_id": "uuid",
      "name": "string",
      "created_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-12-31T00:00:00Z",
      "last_used_at": "2024-06-01T00:00:00Z"
    }
  ]
}
```

### DELETE /tenants/{tenantId}/api-keys/{keyId}
**Auth:** X-Admin-Key or X-API-Key

Response 204: No content (success)
Response 404: Key or tenant not found
Response 403: Key doesn't belong to tenant

## Database
Table: `api_keys` (already exists)

## Files to Modify
- `internal/controlplane/db/store.go` - Add interface methods
- `internal/controlplane/db/postgres.go` - Implement DB queries
- `internal/controlplane/api/server.go` - Add HTTP handlers
- `internal/controlplane/db/memory.go` - Implement for in-memory store (tests)

## Testing
- Unit tests for DB layer
- Integration tests for HTTP handlers
- Test authorization (tenant isolation)

## Definition of Done
- [ ] Code implemented
- [ ] Tests passing (`go test ./...`)
- [ ] `go vet ./...` clean
- [ ] Manual test with curl

## Estimated Effort
4-6 hours
