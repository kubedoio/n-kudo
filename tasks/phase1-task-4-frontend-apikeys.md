# Phase 1 Task 4: Frontend API Key Management

## Task Description
Implement frontend hooks and UI for API key management (list, create, revoke).

## Prerequisites
- Phase 1 Task 1 (Backend endpoints) must be complete

## Acceptance Criteria
- [ ] `useAPIKeys(tenantId)` hook that fetches real API keys
- [ ] `useCreateAPIKey()` mutation hook
- [ ] `useRevokeAPIKey()` mutation hook
- [ ] `CreateAPIKeyModal` component with form
- [ ] TenantDetail API Keys tab uses real data (no mocks)
- [ ] "Create API Key" button opens modal and works
- [ ] "Revoke" button works with confirmation
- [ ] Toast notifications for success/error

## Files to Create/Modify

### New Files
- `frontend/src/api/hooks/useAPIKeys.ts` - Query hook
- `frontend/src/api/hooks/useCreateAPIKey.ts` - Mutation hook
- `frontend/src/api/hooks/useRevokeAPIKey.ts` - Mutation hook
- `frontend/src/pages/Admin/CreateAPIKeyModal.tsx` - Modal component

### Modified Files
- `frontend/src/api/hooks.ts` - Re-export new hooks
- `frontend/src/api/types.ts` - Add APIKey type (if not exists)
- `frontend/src/pages/Admin/TenantDetail.tsx` - Replace mock data

## API Integration

```typescript
// GET /tenants/{tenantId}/api-keys
interface APIKey {
  id: string;
  tenant_id: string;
  name: string;
  created_at: string;
  expires_at: string;
  last_used_at: string | null;
}

// POST /tenants/{tenantId}/api-keys (already exists)
// DELETE /tenants/{tenantId}/api-keys/{keyId} (from Task 1)
```

## UI Flow

### API Keys Tab
1. Load and display table of API keys
2. Each row shows: Name, Created, Expires, Last Used, Actions
3. Actions: Revoke button

### Create API Key Flow
1. Click "Create API Key" button
2. Modal opens with form (name, expires_in_seconds)
3. Submit creates key
4. **CRITICAL:** Display the actual API key once (after creation)
5. Store in state, show copy button
6. Close modal, refresh list

### Revoke Flow
1. Click "Revoke" button
2. Confirmation dialog: "Are you sure? This cannot be undone."
3. On confirm: call delete endpoint
4. Refresh list
5. Toast: "API key revoked"

## Testing
- Component renders with real data
- Create flow works end-to-end
- Revoke with confirmation works
- Error states handled

## Definition of Done
- [ ] No references to `mockAPIKeys`
- [ ] All API operations work
- [ ] User can create and revoke keys
- [ ] Build passes (`npm run build`)

## Estimated Effort
6-8 hours
