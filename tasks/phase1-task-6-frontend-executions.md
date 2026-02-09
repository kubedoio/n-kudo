# Phase 1 Task 6: Frontend Execution History Integration

## Task Description
Replace mock plans data with real execution history and add status polling.

## Prerequisites
- Phase 1 Task 3 (Backend endpoint) must be complete

## Acceptance Criteria
- [ ] `useExecutions(siteId, filters)` hook
- [ ] SiteDashboard Plans tab uses real data
- [ ] Remove `mockPlans` from SiteDashboard
- [ ] Add polling for execution status updates (5-second interval)
- [ ] Execution status badges show correct state
- [ ] Remove `generateMockLogs()` fallback from ExecutionLogViewer

## Files to Create/Modify

### New Files
- `frontend/src/api/hooks/useExecutions.ts`

### Modified Files
- `frontend/src/api/hooks.ts` - Re-export hook
- `frontend/src/pages/Tenant/SiteDashboard.tsx` - Replace mock data
- `frontend/src/pages/Tenant/ExecutionLogViewer.tsx` - Remove mock fallback

## API Integration

```typescript
// GET /sites/{siteId}/executions?status=SUCCEEDED,FAILED&limit=50
interface Execution {
  id: string;
  plan_id: string;
  operation_id: string;
  operation_type: 'CREATE' | 'START' | 'STOP' | 'DELETE';
  state: 'PENDING' | 'IN_PROGRESS' | 'SUCCEEDED' | 'FAILED';
  vm_id: string;
  error_code: string | null;
  error_message: string | null;
  created_at: string;
  updated_at: string;
}
```

## Polling Implementation

```typescript
const { data: executions, refetch } = useExecutions(siteId, {
  status: 'PENDING,IN_PROGRESS',
});

// Poll for updates
useEffect(() => {
  const interval = setInterval(() => {
    refetch();
  }, 5000);
  return () => clearInterval(interval);
}, [refetch]);
```

## UI Changes

### Plans Tab
- Display real executions in table
- Show: ID, Version, Status, Created, Actions
- "View Logs" button opens ExecutionLogViewer
- Auto-refresh when new plans are applied

### Status Badges
```typescript
const statusMap = {
  PENDING: { variant: 'warning', label: 'Pending' },
  IN_PROGRESS: { variant: 'info', label: 'Running' },
  SUCCEEDED: { variant: 'success', label: 'Succeeded' },
  FAILED: { variant: 'error', label: 'Failed' },
};
```

## ExecutionLogViewer Changes
- Remove `generateMockLogs()` function
- Remove fallback to mock data
- Show error state if API fails
- Show empty state if no logs

## Definition of Done
- [ ] Real execution data displayed
- [ ] Polling works (updates every 5s)
- [ ] No mock data references
- [ ] Logs viewer doesn't use mock fallback
- [ ] Build passes

## Estimated Effort
5-6 hours
