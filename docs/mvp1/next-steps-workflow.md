# n-kudo MVP-1 Next Steps Workflow

## Current Status Summary

### ✅ Completed
- **Control Plane Core**: HTTP/TLS server, tenant/site management, enrollment tokens
- **Database Layer**: Postgres migrations, repositories for all core entities
- **Edge Agent**: Static binary with enroll, heartbeat, hostfacts, plan execution
- **Cloud Hypervisor Provider**: Full VM lifecycle (create/start/stop/delete)
- **NetBird Integration**: Join, status checks, probe mechanisms
- **Security**: mTLS enrollment, certificate issuance, ephemeral/dev PKI modes
- **Idempotency**: Per-action execution cache, plan deduplication
- **Tests**: Unit tests for all major packages, basic integration test
- **Demo**: Full end-to-end demo script (`demo.sh`)

### ⚠️ Partially Implemented
- **gRPC/proto**: Contracts defined but runtime server uses HTTP/JSON
- **Integration Tests**: Only 1 of 5 mandatory test suites implemented

### ❌ Not Implemented
- CI/CD pipeline
- Release automation (multi-arch binaries, container images)
- Mock NetBird/Cloud Hypervisor for testing
- Dashboard/UI
- Worker binary (mentioned in structure)
- Multi-host site orchestration (post-MVP)

---

## Recommended Next Steps Workflow

### Phase 1: Stabilize & Harden (Week 1)

#### 1.1 Run the Demo and Document Issues
```bash
# Run the full demo
sudo -E ./demo.sh

# Check what works and what doesn't
docker compose -f deployments/docker-compose.yml logs backend
cat .demo/mvp1/work/*.json
```

**Goal**: Confirm the MVP works end-to-end on your target environment.

#### 1.2 Implement Missing Integration Tests
Create the 4 remaining mandatory test suites from `docs/mvp1/acceptance-and-test-plan.md`:

| Test Suite | File | Description |
|------------|------|-------------|
| `TestEnrollAndHeartbeat` | ✅ `tests/integration/fake_controlplane_test.go` | Already exists |
| `TestApplyPlanCreateStartStopDelete` | 📝 `tests/integration/vm_lifecycle_test.go` | VM lifecycle through plan execution |
| `TestApplyPlanIdempotency` | 📝 `tests/integration/idempotency_test.go` | Duplicate plan handling |
| `TestCrossTenantIsolationRejected` | 📝 `tests/integration/tenant_isolation_test.go` | Security boundary validation |
| `TestExecutionLogStreaming` | 📝 `tests/integration/log_streaming_test.go` | End-to-end log flow |

**Priority**: HIGH - These are required for MVP sign-off.

#### 1.3 Create Mock Infrastructure
```
tests/integration/mocks/
  ├── mock_netbird.go          # NetBird management API mock
  └── mock_cloudhypervisor.go   # CH API socket mock
```

**Purpose**: Enable integration tests without requiring actual NetBird/CH binaries.

---

### Phase 2: CI/CD & Release Automation (Week 2)

#### 2.1 GitHub Actions CI Pipeline
```
.github/workflows/
  ├── ci.yml                    # Lint, test, build on PR
  └── release.yml               # Release builds on tag
```

**Pipeline Stages**:
1. Lint (`golangci-lint`, `go vet`)
2. Unit tests (`go test ./...`)
3. Integration tests (with ephemeral Postgres)
4. Build artifacts:
   - `nkudo-edge-linux-amd64`
   - `nkudo-edge-linux-arm64`
   - Control-plane container image
5. Sign and publish

#### 2.2 Build Scripts
```
scripts/
  ├── build-edge.sh             # Multi-arch static binary build
  ├── build-control-plane.sh    # Container image build
  └── release.sh                # Full release process
```

#### 2.3 Packaging
- Tarball with binary + systemd unit + sample config
- Container image for control-plane
- Helm chart (optional but recommended)

---

### Phase 3: Production Readiness (Week 3)

#### 3.1 Operational Tooling
```
scripts/
  ├── health-check.sh           # Control-plane health probe
  ├── agent-debug.sh            # Collect agent diagnostics
  └── tenant-offboard.sh        # GDPR-compliant tenant deletion
```

#### 3.2 Runbooks (docs/runbooks/)
- Enrollment failure diagnosis
- NetBird join failure recovery
- Cloud Hypervisor command failures
- Certificate rotation procedures
- Database backup/restore

#### 3.3 Monitoring & Alerting
Control-plane metrics endpoints:
- `GET /metrics` (Prometheus format)
- Heartbeat lag per site
- Plan execution failure rate
- Enrollment success rate

#### 3.4 Security Hardening
- [ ] Implement `REQUIRE_PERSISTENT_PKI=true` validation
- [ ] Certificate revocation list (CRL) support
- [ ] Rate limiting on enrollment endpoints
- [ ] Audit log integrity verification

---

### Phase 4: gRPC Migration (Optional, Post-MVP)

#### 4.1 Implement gRPC Server
```
internal/controlplane/api/grpc/
  ├── server.go
  ├── enroll.go
  ├── heartbeat.go
  └── logs.go
```

#### 4.2 Update Edge Agent
- Add gRPC client alongside HTTP
- Fallback mechanism (gRPC -> HTTP)

**Note**: Current HTTP/JSON works for MVP-1. gRPC was in architecture but not required for initial release.

---

## Immediate Action Items (This Week)

### 1. Verify Demo Works
```bash
# Clean environment
sudo rm -rf .demo/mvp1
docker compose -f deployments/docker-compose.yml down -v

# Run demo
sudo -E ./demo.sh 2>&1 | tee demo-output.log
```

### 2. Check Prerequisites
Ensure your environment has:
- [ ] Docker + Docker Compose
- [ ] Go 1.23+
- [ ] cloud-hypervisor binary
- [ ] cloud-localds (or genisoimage/mkisofs)
- [ ] iproute2 (for TAP/bridge)
- [ ] jq, curl

### 3. Review Configuration
Key environment variables for production:
```bash
# Control-plane
export REQUIRE_PERSISTENT_PKI=true
export CA_CERT_FILE=/etc/nkudo/ca.crt
export CA_KEY_FILE=/etc/nkudo/ca.key
export SERVER_CERT_FILE=/etc/nkudo/server.crt
export SERVER_KEY_FILE=/etc/nkudo/server.key

# Edge Agent (after enrollment)
/usr/local/bin/nkudo-edge run \
  --control-plane https://api.nkudo.io \
  --state-dir /var/lib/nkudo-edge/state \
  --pki-dir /var/lib/nkudo-edge/pki
```

---

## Decision Points

### Decision 1: Is HTTP/JSON sufficient for MVP-1?
- **Option A**: Keep HTTP/JSON (current), defer gRPC to MVP-2
- **Option B**: Implement gRPC server before release

**Recommendation**: Option A - HTTP/JSON works and meets all acceptance criteria.

### Decision 2: Is the demo environment sufficient for initial customers?
- **Option A**: Use current manual deployment
- **Option B**: Build automated installer (`scripts/install-edge.sh`)

**Recommendation**: Option B - Create a one-line installer for customer onboarding.

### Decision 3: What monitoring is required for MVP?
- **Option A**: Basic health checks only
- **Option B**: Full Prometheus metrics + alerting rules

**Recommendation**: Option B - Metrics are essential for SaaS operations.

---

## File Checklist for Completion

### Core Implementation
- [x] `cmd/control-plane/main.go`
- [x] `cmd/edge/main.go`
- [x] `internal/controlplane/api/server.go`
- [x] `internal/controlplane/db/postgres.go`
- [x] `internal/edge/enroll/client.go`
- [x] `internal/edge/executor/executor.go`
- [x] `internal/edge/providers/cloudhypervisor/provider.go`

### Tests
- [x] Unit tests for all packages
- [x] `tests/integration/fake_controlplane_test.go`
- [ ] `tests/integration/vm_lifecycle_test.go`
- [ ] `tests/integration/idempotency_test.go`
- [ ] `tests/integration/tenant_isolation_test.go`
- [ ] `tests/integration/log_streaming_test.go`

### Deployment
- [x] `deployments/docker-compose.yml`
- [ ] `.github/workflows/ci.yml`
- [ ] `.github/workflows/release.yml`
- [ ] `scripts/build-edge.sh`
- [ ] `scripts/install-edge.sh`

### Documentation
- [x] `README.md`
- [x] `docs/mvp1/architecture.md`
- [x] `docs/mvp1/acceptance-and-test-plan.md`
- [ ] `docs/runbooks/enrollment-failure.md`
- [ ] `docs/runbooks/netbird-troubleshooting.md`

---

## Success Criteria for MVP-1

Per `docs/mvp1/acceptance-and-test-plan.md`:

1. ✅ Site onboarding with enrollment tokens works
2. ✅ Heartbeat and host facts flow correctly
3. ⚠️ NetBird integration (basic - partially tested)
4. ✅ Cloud Hypervisor microVM lifecycle works
5. ✅ Execution status and logs are captured
6. ⚠️ Tenant isolation (needs dedicated test)
7. ⚠️ Audit events coverage (verify all actions)

**Exit Criteria**:
- [ ] All mandatory integration suites pass in CI
- [ ] E2E smoke test passes twice consecutively
- [ ] No P0/P1 defects open for enrollment, plan execution, or tenant isolation

---

*Generated based on repository analysis as of February 8, 2026*
