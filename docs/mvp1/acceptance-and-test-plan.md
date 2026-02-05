# MVP-1 Acceptance Criteria and Test Plan

## Acceptance criteria

### A. Site onboarding
- Given a valid one-time enrollment token, when `nkudo-agent enroll` is executed, then the control-plane creates exactly one `agents` record and returns credentials.
- Given the same token is reused, enrollment is rejected with `ALREADY_USED`.
- Given an expired token, enrollment is rejected with `TOKEN_EXPIRED`.
- Dashboard shows site status `ONLINE` within 30 seconds of successful enroll.

### B. Heartbeat and host facts
- Agent sends heartbeat every 15s (+/- jitter), and control-plane updates `sites.last_heartbeat_at` and `hosts.last_facts_at`.
- Host facts include CPU/memory/storage totals and KVM/Cloud Hypervisor capability flags.
- If heartbeats stop for >45s, site transitions to `OFFLINE`.

### C. NetBird integration (MVP basic)
- Agent can join NetBird mesh using issued credentials.
- Heartbeat reports `netbird_status.connected=true` when peer is connected.
- Control-plane stores peer metadata and exposes it in status query.

### D. Cloud Hypervisor microVM lifecycle
- `ApplyPlan` with `CREATE` creates a VM record and CH instance.
- `START` changes state to `RUNNING`; `STOP` changes to `STOPPED`; `DELETE` removes VM runtime and sets terminal record state.
- Duplicate `ApplyPlan` with same `idempotency_key` returns same `plan_id` and does not create duplicate operations.

### E. Execution status and logs
- For each plan operation, an `executions` row is created and transitions to terminal state (`SUCCEEDED`/`FAILED`).
- Agent streams logs; dashboard can query recent logs for an execution.
- Failed operations include non-empty `error_code` and `error_message`.

### F. Tenant isolation and auditability
- Tenant A cannot query or mutate resources of Tenant B.
- Enrollment, ApplyPlan, VM lifecycle transitions, and auth failures create `audit_events` entries.
- PII/data minimization check: no guest workload payload contents are persisted.

## Local dev environment instructions

Prerequisites:
- Linux/macOS dev machine with Docker, Go 1.22+, `protoc`, and `buf`.
- `cloud-hypervisor` binary available in PATH for runtime tests.

Steps:
1. Start dependencies:
   - `docker compose up -d postgres netbird-mock`
2. Apply DB migrations:
   - `go run ./cmd/control-plane migrate up`
3. Run control-plane locally:
   - `go run ./cmd/control-plane --config ./deployments/dev/control-plane.yaml`
4. Run mock Cloud Hypervisor service (for integration tests):
   - `go test ./tests/integration/mock_cloudhypervisor -run TestServe -v`
5. Run agent against local control-plane:
   - `go run ./cmd/nkudo-agent --config ./deployments/dev/agent.yaml`
6. Trigger enrollment + sample plan via seed script/API client.

## Test strategy

### Unit tests
- Control-plane:
  - enrollment token validation
  - ApplyPlan idempotency logic
  - tenant-scope authorization middleware
- Agent:
  - plan executor step ordering and retry behavior
  - local idempotency cache
  - heartbeat payload builder

### Integration tests (required for MVP sign-off)
- Use `mock_netbird` server to emulate join success/failure and connectivity status.
- Use `mock_cloudhypervisor` binary/API shim to emulate VM operations and injected failures.
- Use ephemeral Postgres per suite (Docker).

Mandatory suites:
1. `TestEnrollAndHeartbeat`
2. `TestApplyPlanCreateStartStopDelete`
3. `TestApplyPlanIdempotency`
4. `TestCrossTenantIsolationRejected`
5. `TestExecutionLogStreaming`

### End-to-end smoke test
- Single script in `tests/e2e/smoke.sh`:
  - Create tenant/user/site
  - Issue enrollment token
  - Enroll agent
  - Wait for site online
  - Apply VM create/start/stop/delete plan
  - Assert final execution states and audit events

### Exit criteria for MVP-1
- All mandatory integration suites pass in CI.
- E2E smoke test passes twice consecutively on a clean environment.
- No P0/P1 defects open for enrollment, plan execution, or tenant isolation.

