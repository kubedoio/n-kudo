# Phase 1 Task 5: Frontend Token History Integration

## Task Description
Integrate real enrollment token history into the frontend.

## Prerequisites
- Phase 1 Task 2 (Backend endpoint) must be complete

## Acceptance Criteria
- [ ] `useEnrollmentTokens(tenantId)` hook
- [ ] Token History tab in TenantDetail uses real data
- [ ] No references to `mockTokenHistory`
- [ ] Copy token functionality works

## Files to Create/Modify

### New Files
- `frontend/src/api/hooks/useEnrollmentTokens.ts`

### Modified Files
- `frontend/src/api/hooks.ts` - Re-export hook
- `frontend/src/api/types.ts` - Add EnrollmentToken type
- `frontend/src/pages/Admin/TenantDetail.tsx` - Replace mock data

## API Integration

```typescript
// GET /tenants/{tenantId}/enrollment-tokens
interface EnrollmentToken {
  id: string;
  site_id: string;
  site_name: string;
  created_at: string;
  expires_at: string;
  consumed: boolean;
  consumed_at: string | null;
  consumed_by_agent_id: string | null;
}
```

## UI Requirements

### Token History Tab
Display table with columns:
- Site (with icon + name)
- Status (badge: Used/Expired/Pending)
- Created (timestamp)
- Expires (timestamp)
- Token (copy button)

### Status Logic
```typescript
if (consumed) return { label: 'Used', variant: 'success' };
if (new Date(expires_at) < new Date()) return { label: 'Expired', variant: 'error' };
return { label: 'Pending', variant: 'warning' };
```

### Copy Token
Note: The actual token string is only available at creation time. For history:
- Show "Copy" button only if token is still pending (not consumed/expired)
- OR show token ID with "Already consumed" message

## Definition of Done
- [ ] Hook implemented and working
- [ ] Real data displayed in UI
- [ ] No mock data references
- [ ] Build passes

## Estimated Effort
3-4 hours
