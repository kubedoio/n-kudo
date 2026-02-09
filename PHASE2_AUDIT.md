# Phase 2 Implementation Audit Report

**Date:** 2026-02-09  
**Auditor:** Kimi Code CLI (Parallel Sub-agents)

---

## Executive Summary

Phase 2 test implementation has **significant issues** that must be fixed before proceeding to Phase 3:

| Severity | Count | Category |
|----------|-------|----------|
| 🔴 Critical | 8 | False definitions, signature mismatches |
| 🟠 High | 12 | Race conditions, unchecked assertions |
| 🟡 Medium | 20+ | Duplications, ignored errors |

---

## 🔴 Critical Issues (Must Fix)

### 1. Frontend: Wrong Import Paths (5 files)
**Files:** All hook test files (`useAPIKeys.test.ts`, `useApplyPlan.test.ts`, etc.)

**Issue:** Tests import from `../../hooks` but hooks are in `../../hooks.ts`
```typescript
// WRONG:
import { useAPIKeys } from '../../hooks'

// CORRECT:
import { useAPIKeys } from '../../hooks.ts'
```

**Impact:** Tests may be testing wrong implementation or failing silently.

---

### 2. Frontend: Wrong Hook Signatures (3 tests)
**Files:** `useApplyPlan.test.ts`, `useCreateAPIKey.test.ts`, `useCreateTenant.test.ts`

**Issue:** Tests pass `{ onSuccess }` directly, but hooks expect TanStack Query `UseMutationOptions`:
```typescript
// WRONG in tests:
useApplyPlan({ onSuccess })

// CORRECT:
useApplyPlan({ 
  onSuccess: (data, variables, context, mutation) => { ... }
})
```

---

### 3. Edge Agent: Mock Signature Mismatches (2 mocks)
**File:** `internal/edge/netbird/mock_netbird.go`

**Issue:** Mock methods don't match real interface signatures:
```go
// Real interface:
func (c Client) Join(ctx context.Context, setupKey, hostname string) error
func (c Client) Status(ctx context.Context) (Status, error)

// Mock (WRONG):
func (m *MockClient) Join() error
func (m *MockClient) Status() (Status, error)
```

**Impact:** Code tested with mocks won't compile against real implementation.

---

### 4. Backend: Unchecked Type Assertions (15+ locations)
**Files:** `enrollment_test.go`, `plan_execution_test.go`, `idempotency_test.go`

**Issue:** Type assertions without ok-check will panic on API changes:
```go
// WRONG:
planID := createResult["plan_id"].(string)

// CORRECT:
planID, ok := createResult["plan_id"].(string)
if !ok {
    t.Fatalf("expected plan_id string, got %T", createResult["plan_id"])
}
```

---

### 5. Edge Agent: Mock Never Fails
**File:** `internal/edge/executor/executor_test.go:365-387`

**Issue:** `fakeFailingProvider` always returns nil, never fails:
```go
func (f *fakeFailingProvider) Create(ctx context.Context, params MicroVMParams) error {
    if params.VMID == "vm-1" {
        return nil // First VM succeeds
    }
    return nil  // BUG: Always returns nil, never fails!
}
```

---

## 🟠 High Severity Issues

### 6. Backend: Ambiguous Status Assertions
**File:** `security_test.go:88-91`

**Issue:** Test accepts two different status codes, masking incorrect behavior:
```go
// WRONG - masks routing vs auth issues:
if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
    t.Fatalf("expected 403 or 404, got %d", resp.StatusCode)
}
```

---

### 7. Edge Agent: Race Conditions (4 locations)
**Files:** `mock_provider_test.go`, `mock_provider.go`

**Issues:**
- `t.Errorf` called from goroutines (unsafe)
- Direct map access without mutex
- Unsynchronized state transitions

---

### 8. Backend: Timing-Dependent Test
**File:** `enrollment_test.go:239-240`

**Issue:** `time.Sleep(2 * time.Second)` is non-deterministic and flaky on slow CI.

---

### 9. Frontend: Fragile DOM Queries
**File:** `SiteDashboard.test.tsx:117-119`

**Issue:** Uses `nextElementSibling` which breaks on UI changes:
```typescript
const totalVms = screen.getByText('Total VMs').nextElementSibling
```

---

### 10. Backend: Weak Idempotency Test
**File:** `idempotency_test.go:98-100`

**Issue:** Uses `t.Logf` instead of `t.Fatalf` - test **always passes** even if assertion fails.

---

## 🟡 Medium Severity Issues

### 11. Duplications

**Backend:**
- Control plane setup duplicated across 6 files
- List helper functions duplicated (3x)
- HTTP API helpers duplicated

**Frontend:**
- `createWrapper` function duplicated in 5 files
- Toast store mocks duplicated in 2 files
- Same test patterns repeated

**Edge Agent:**
- `fakeProvider` duplicates `MockProvider`
- State test setup pattern repeated 15+ times
- Same CRUD test patterns

---

### 12. Ignored Errors
**Files:** `state/state_test.go`, `executor/executor_test.go`

**Issue:** Multiple errors ignored with `_ = ...` pattern, masking failures.

---

### 13. Mock Data Type Mismatches
**File:** `frontend/src/test/mocks/handlers.ts`

**Issue:** Mock data missing required fields or wrong structure.

---

## Recommendations

### Immediate Actions (Before Phase 3)

1. **Fix critical signature mismatches** in `mock_netbird.go`
2. **Fix import paths** in all frontend hook tests
3. **Fix hook signatures** in frontend tests
4. **Add safe type assertions** in backend integration tests
5. **Fix fakeFailingProvider** to actually fail
6. **Fix race conditions** in edge agent tests

### Code Quality Improvements (Can defer)

7. Extract shared helpers to reduce duplication
8. Replace fragile DOM queries with test IDs
9. Add proper error checking instead of ignoring
10. Fix timing-dependent tests with retry loops

---

## Impact Assessment

| Component | Status | Blocker for Phase 3? |
|-----------|--------|----------------------|
| Backend Integration Tests | ⚠️ Needs fixes | Yes - false positives risk |
| Frontend Unit Tests | 🔴 Broken | Yes - wrong imports/signatures |
| E2E Tests | ✅ OK | No |
| Edge Agent Tests | 🔴 Broken | Yes - signature mismatches |

---

## Fix Priority

**Priority 1 (Fix Now):**
1. mock_netbird.go signatures
2. Frontend hook test imports
3. Frontend hook test signatures
4. fakeFailingProvider logic

**Priority 2 (Fix Soon):**
5. Backend type assertions
6. Race conditions
7. Weak test assertions

**Priority 3 (Nice to have):**
8. Code deduplication
9. Test infrastructure improvements
