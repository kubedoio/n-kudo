# N-Kudo Implementation Roadmap

This document outlines the planned phases for completing the n-kudo MVP and beyond.

## Current Status (MVP-1)

✅ **Completed:**
- Tenant, Site, API Key management (backend)
- Agent enrollment with mTLS
- Heartbeat ingestion
- Plan execution with idempotency
- Cloud Hypervisor provider (VM CRUD)
- Basic frontend with real API integration
- Execution log streaming

⚠️ **Partial/Needs Work:**
- Frontend uses mock data for API Keys, Token History, Plans
- Missing integration tests
- No CI/CD pipeline
- gRPC proto defined but HTTP/JSON runtime only

---

## Phase 1: Frontend Production Readiness (Week 1-2) ✅ COMPLETE

**Goal:** Replace all mock data with real API calls, complete essential UI features.

### 1.1 API Key Management (Frontend) ✅
- [x] Create `GET /tenants/{tenantId}/api-keys` endpoint
- [x] Create `DELETE /tenants/{tenantId}/api-keys/{keyId}` endpoint  
- [x] Create `useAPIKeys()` hook
- [x] Create `useRevokeAPIKey()` mutation hook
- [x] Create `CreateAPIKeyModal` component
- [x] Wire up "Revoke" button in TenantDetail API Keys tab
- [x] Remove `mockAPIKeys` from TenantDetail.tsx

**Acceptance Criteria:**
- ✅ Admin can view real API keys list
- ✅ Admin can create new API key
- ✅ Admin can revoke existing API key
- ✅ No mock data remains

### 1.2 Token History (Frontend) ✅
- [x] Create `GET /tenants/{tenantId}/enrollment-tokens` endpoint
- [x] Create `useEnrollmentTokens()` hook
- [x] Update Token History tab in TenantDetail
- [x] Remove `mockTokenHistory` from TenantDetail.tsx

**Acceptance Criteria:**
- ✅ Admin can view all issued enrollment tokens
- ✅ Shows correct status (used/pending/expired)
- ✅ No mock data remains

### 1.3 Execution/Plan History (Frontend) ✅
- [x] Create `GET /sites/{siteId}/executions` endpoint (with filters)
- [x] Create `useExecutions()` hook
- [x] Replace `mockPlans` in SiteDashboard.tsx
- [x] Add polling for execution status updates
- [x] Remove `generateMockLogs()` fallback from ExecutionLogViewer

**Acceptance Criteria:**
- ✅ Site dashboard shows real execution history
- ✅ Execution status updates in real-time (polling)
- ✅ Logs viewer only shows real data

### 1.4 UI Polish & Bug Fixes ✅
- [x] Add loading states to all async operations
- [x] Add error boundaries
- [x] Fix TypeScript `any` types in table columns
- [x] Delete legacy pages (AdminTenants.tsx, SitesList.tsx, SiteDashboard.tsx at root)

---

## Phase 2: Testing & Quality (Week 2-3) ✅ COMPLETE

**Goal:** Achieve >80% test coverage, add integration tests.

### 2.1 Backend Integration Tests ✅
- [x] `TestApplyPlanCreateStartStopDelete` - Full VM lifecycle
- [x] `TestApplyPlanIdempotency` - Ensure plans are idempotent
- [x] `TestCrossTenantIsolationRejected` - Security boundary test
- [x] `TestExecutionLogStreaming` - Log ingestion and retrieval
- [x] `TestAgentEnrollmentFlow` - End-to-end enrollment

**Result:** 26 integration tests passing

### 2.2 Frontend Testing ✅
- [x] Setup React Testing Library
- [x] Unit tests for API hooks (14 tests)
- [x] Component tests for critical UI (14 tests)
- [x] E2E tests with Playwright (critical paths)

**Result:** 28 frontend tests passing

### 2.3 Edge Agent Tests ✅
- [x] Mock Cloud Hypervisor for testing
- [x] Mock NetBird for testing
- [x] Unit tests for executor action handlers
- [x] Integration test for enrollment + plan execution

**Result:** 40+ edge agent tests passing, >80% coverage

---

## Phase 3: Edge Agent Enhancements (Week 3-4) ✅ COMPLETE

**Goal:** Add new commands, observability, and action types.

### 3.1 New Agent Commands ✅
- [x] `nkudo status` - Show enrollment and connection status
- [x] `nkudo check` - Pre-flight requirements check
- [x] `nkudo unenroll` - Clean removal from site
- [x] `nkudo renew` - Manual certificate renewal

### 3.2 Observability ✅
- [x] Prometheus metrics endpoint (`:9090/metrics`)
- [x] Structured JSON logging
- [x] Configurable log levels
- [x] Execution duration tracking

### 3.3 New Action Types ✅
- [x] `MicroVMPause` / `MicroVMResume` - VM state management
- [x] `MicroVMSnapshot` - Create VM snapshots
- [x] `CommandExecute` - Execute arbitrary commands on host

---

## Phase 4: Security Hardening (Week 4-5)

**Goal:** Production-grade security.

### 4.1 Certificate Management
- [ ] Automatic certificate rotation before expiry
- [ ] Certificate Revocation List (CRL) support
- [ ] `REQUIRE_PERSISTENT_PK=true` validation enforcement

### 4.2 Rate Limiting & Protection
- [ ] Rate limiting on enrollment endpoint
- [ ] API key attempt limiting
- [ ] Audit log integrity verification

### 4.3 Secret Management
- [ ] Integration with external secret stores (Vault, etc.)
- [ ] Encrypted local state at rest

---

## Phase 3: Edge Agent Enhancements (Week 3-4)

**Goal:** Improve agent observability, add missing commands.

### 3.1 New Agent Commands
- [ ] `nkudo status` - Show enrollment status, connection health, cert expiry
- [ ] `nkudo check` - Pre-flight check (KVM, CH, bridges, permissions)
- [ ] `nkudo unenroll` - Clean removal from site
- [ ] `nkudo renew` - Manual certificate renewal

### 3.2 Observability
- [ ] Structured JSON logging
- [ ] Configurable log levels
- [ ] Prometheus metrics endpoint (`/metrics` on agent)
- [ ] Execution duration tracking

### 3.3 New Action Types
- [ ] `MicroVMPause` / `MicroVMResume` - VM state management
- [ ] `CommandExecute` - Execute arbitrary commands on host
- [ ] `FileWrite` / `FileDelete` - File management

---

## Phase 4: Security Hardening (Week 4-5)

**Goal:** Production-grade security.

### 4.1 Certificate Management
- [ ] Automatic certificate rotation before expiry
- [ ] Certificate Revocation List (CRL) support
- [ ] `REQUIRE_PERSISTENT_PK=true` validation enforcement

### 4.2 Rate Limiting & Protection
- [ ] Rate limiting on enrollment endpoint
- [ ] API key attempt limiting
- [ ] Audit log integrity verification

### 4.3 Secret Management
- [ ] Integration with external secret stores (Vault, etc.)
- [ ] Encrypted local state at rest

---

## Phase 5: Advanced Features (Week 5-6)

**Goal:** Additional providers, networking, scalability.

### 5.1 Additional VM Providers
- [ ] Firecracker provider
- [ ] QEMU provider
- [ ] Provider auto-detection

### 5.2 Networking
- [ ] Multiple network interfaces per VM
- [ ] VXLAN/overlay network support
- [ ] Bandwidth limiting/QoS

### 5.3 gRPC Runtime
- [ ] Implement gRPC server alongside HTTP/JSON
- [ ] gRPC client for edge agent (optional)

---

## Phase 6: DevOps & Deployment (Week 6-7)

**Goal:** Production deployment readiness.

### 6.1 CI/CD Pipeline
- [ ] GitHub Actions workflow for tests
- [ ] Automated Docker image builds
- [ ] Release automation with changelogs
- [ ] Multi-arch builds (amd64, arm64)

### 6.2 Installation & Packaging
- [ ] `scripts/install-edge.sh` - One-line agent installer
- [ ] Systemd service files
- [ ] APT/YUM package repositories
- [ ] Helm chart for control-plane

### 6.3 Documentation
- [ ] API reference (auto-generated from OpenAPI)
- [ ] Deployment runbook
- [ ] Troubleshooting guide
- [ ] Architecture decision records (ADRs)

---

## Quick Wins (Can be done anytime)

These are smaller tasks that can be picked up between larger phases:

- [ ] Fix Sidebar navigation links (networks, users, settings pages)
- [ ] Add 404 page
- [ ] Add global search functionality
- [ ] VM console access (VNC/serial)
- [ ] Bulk VM operations
- [ ] Execution retry functionality
- [ ] Tenant-level usage quotas

---

## Decision Points

### 1. gRPC vs HTTP/JSON
**Current:** Proto defined, HTTP/JSON runtime only
**Options:**
- Option A: Implement gRPC server (Phase 5)
- Option B: Remove proto files, focus on HTTP/JSON
- **Recommendation:** Option B for MVP, revisit for v2

### 2. Edge Module Structure
**Current:** Nested module in `edge/` not used by main code
**Options:**
- Option A: Migrate edge agent to use nested module
- Option B: Remove nested module
- **Recommendation:** Option B to reduce confusion

### 3. Frontend State Management
**Current:** TanStack Query + Zustand (minimal)
**Options:**
- Option A: Add Redux Toolkit for complex state
- Option B: Keep minimal, add only when needed
- **Recommendation:** Option B

---

## Success Metrics

| Phase | Metric | Target |
|-------|--------|--------|
| 1 | Mock data eliminated | 0 mock data sources |
| 2 | Test coverage | >80% |
| 2 | Integration tests | 5+ E2E scenarios |
| 3 | New agent commands | 4+ commands |
| 4 | Security audit | 0 critical issues |
| 5 | Additional providers | 2+ providers |
| 6 | CI/CD | Full automation |

---

## Task Dependencies

```
Phase 1.1 (API Keys Backend) ──┐
                              ├──→ Phase 1.1 (API Keys Frontend)
Phase 1.1 (API Keys Frontend) ──┘

Phase 1.2 (Token History) ──→ Phase 1 (Complete)

Phase 1.3 (Executions) ─────→ Phase 1 (Complete)

Phase 1 ────────────────────→ Phase 2 (Testing)

Phase 2 ────────────────────→ Phase 3 (Agent)
                          └─→ Phase 4 (Security)
                          
Phase 3 ────────────────────→ Phase 5 (Advanced)

All Phases ─────────────────→ Phase 6 (DevOps)
```

---

*Last updated: 2026-02-09*
