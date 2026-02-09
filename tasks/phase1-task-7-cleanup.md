# Phase 1 Task 7: Frontend Cleanup

## Task Description
Remove legacy code, mock data, and clean up the frontend codebase.

## Acceptance Criteria
- [ ] Delete legacy pages (AdminTenants.tsx, SitesList.tsx, SiteDashboard.tsx from root pages/)
- [ ] Fix TypeScript `any` types in table columns
- [ ] Add error boundaries
- [ ] Fix unused imports

## Files to Delete
- `frontend/src/pages/AdminTenants.tsx` - Legacy, uses mock data
- `frontend/src/pages/SitesList.tsx` - Legacy, uses mock data
- `frontend/src/pages/SiteDashboard.tsx` - Legacy, uses mock data

## Files to Modify

### Fix TypeScript Types
Files with `any` types to fix:
- `frontend/src/pages/Tenant/SiteDashboard.tsx` - vmColumns, hostColumns, planColumns
- `frontend/src/pages/Admin/TenantDetail.tsx` - siteColumns, keyColumns, tokenColumns
- `frontend/src/pages/Admin/TenantsList.tsx` - columns

Example fix:
```typescript
// Before
const vmColumns: TableColumn<any>[] = [

// After  
const vmColumns: TableColumn<MicroVM>[] = [
```

### Add Error Boundaries
Create `frontend/src/components/ErrorBoundary.tsx`:
```typescript
class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean; error?: Error }
> {
  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }
  
  render() {
    if (this.state.hasError) {
      return <ErrorFallback error={this.state.error} />;
    }
    return this.props.children;
  }
}
```

Wrap routes in App.tsx:
```typescript
<Route path="/admin/tenants" element={
  <ErrorBoundary><TenantsList /></ErrorBoundary>
} />
```

### Fix Unused Imports
Files to check:
- `frontend/src/components/Header.tsx`
- `frontend/src/components/Sidebar.tsx`
- `frontend/src/api/hooks.ts`

Run ESLint to find unused imports:
```bash
cd frontend && npm run lint
```

## Definition of Done
- [ ] Legacy files deleted
- [ ] No TypeScript `any` types in table columns
- [ ] Error boundaries in place
- [ ] No unused imports
- [ ] Build passes
- [ ] App works correctly

## Estimated Effort
3-4 hours
