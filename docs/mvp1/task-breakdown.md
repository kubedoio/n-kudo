# MVP-1 Task Breakdown (4 Engineering Agents)

## Schedule assumption
- Total duration: 2-3 weeks.
- Team size: 4 engineers working in parallel with dependency gates.

## Agent 1: Control-plane Core + API
Owner scope:
- Implement gRPC services: `Enroll`, `Heartbeat`, `ApplyPlan`, `GetStatus`, `StreamLogs` ingest.
- Implement auth middleware and tenant scoping.
- Implement plan idempotency (`idempotency_key`) and status transitions.
- Expose dashboard-ready status read model.

Deliverables:
- `cmd/control-plane` service running with API endpoints.
- Request/response validation and error model.
- API integration tests for key flows.

Dependencies:
- Depends on Agent 3 DB migration contracts.
- Receives generated protobuf from shared API contract.

## Agent 2: Edge Agent Runtime + Cloud Hypervisor
Owner scope:
- Build `nkudo-edge` static binary and systemd service.
- Enrollment client and mTLS credential storage/refresh.
- Heartbeat loop + host facts collector.
- Plan executor and Cloud Hypervisor provider (`create/start/stop/delete`).
- Log streaming client.

Deliverables:
- `cmd/edge` with config and retry/backoff behavior.
- Local idempotency cache and crash-safe resume.
- Integration tests against mock Cloud Hypervisor.

Dependencies:
- Depends on Agent 1 APIs being available in dev/staging.
- Depends on Agent 4 mock infrastructure for integration tests.

## Agent 3: Data Layer + Audit + Tenancy Guardrails
Owner scope:
- Author and validate Postgres migration (`0001_mvp1.sql`).
- Repository layer for tenants/sites/hosts/agents/microvms/plans/executions/audit_events.
- Query patterns for dashboard status and execution history.
- Audit event coverage for all critical actions.

Deliverables:
- Migration + seed scripts.
- DB transaction boundaries and index tuning.
- Tenant isolation test cases at repository layer.

Dependencies:
- Supports Agent 1 and Agent 2 with stable schema and repository interfaces.

## Agent 4: NetBird Integration + Test/Release Infrastructure
Owner scope:
- Implement NetBird adapter in control-plane and agent-side join/status integration.
- Build `mock_netbird` and `mock_cloudhypervisor` harnesses.
- Own CI pipelines, integration suites, packaging scripts, release manifest/signing.
- Define rollout and rollback scripts/checklists.

Deliverables:
- Passing integration suites in CI.
- Release artifacts for agent and control-plane.
- E2E smoke script for MVP sign-off.

Dependencies:
- Depends on Agent 1/2 for API + runtime hooks.
- Unblocks final sign-off for all agents.

## Dependency graph (high level)
1. Shared contracts freeze (`proto` + DB schema): Day 1-2 (Agents 1 + 3).
2. Parallel implementation:
   - Agent 1 control-plane API
   - Agent 2 agent runtime
   - Agent 4 mocks and CI harness
3. Integration merge point: Day 8-10 (all agents).
4. Stabilization, bug fixes, release prep: Day 11-15.

## MVP critical path
- Contract freeze -> enrollment end-to-end -> heartbeat/status -> apply plan lifecycle -> integration tests green -> release packaging.
