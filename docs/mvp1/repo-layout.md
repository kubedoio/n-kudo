# MVP-1 Repo Layout and Module Boundaries

## Proposed monorepo structure

```text
n-kudo/
  api/
    proto/controlplane/v1/controlplane.proto
  cmd/
    control-plane/main.go
    nkudo-agent/main.go
  internal/
    controlplane/
      api/                  # gRPC/HTTP handlers, auth middleware, request validation
      enrollment/           # token validation, cert bootstrap/rotation
      plans/                # apply-plan logic, scheduler, idempotency checks
      ingest/               # heartbeat processing, host facts upsert, execution updates
      status/               # site/host/microvm query composition
      netbird/              # SaaS-side adapter for NetBird API
      audit/                # audit event writer
      tenancy/              # tenant boundary enforcement helpers
    agent/
      enroll/               # bootstrap + credential persistence
      controlclient/        # gRPC client and retry/backoff
      heartbeat/            # periodic reporting loop
      facts/                # host inventory collection
      planner/              # pull pending plans and dispatch operations
      executor/             # operation execution state machine
      logs/                 # execution log buffering + streaming
      store/                # local state (idempotency cache, plan checkpoints)
      runtime/
        cloudhypervisor/    # Cloud Hypervisor provider module
      network/
        netbird/            # NetBird integration module
  db/
    migrations/0001_mvp1.sql
  deployments/
    systemd/nkudo-agent.service
    packaging/
      Dockerfile.control-plane
      release-agent.sh
  tests/
    integration/
      fixtures/
      mock_netbird/
      mock_cloudhypervisor/
    e2e/
  docs/mvp1/
    architecture.md
    acceptance-and-test-plan.md
    release-plan.md
    task-breakdown.md
```

## Module boundaries

- `internal/controlplane/api`
  - Responsibility: expose APIs; enforce authN/authZ and tenant context; map transport DTOs to domain commands.
  - No direct DB SQL outside repositories.

- `internal/controlplane/plans`
  - Responsibility: create immutable plan records, validate action sequences, enforce idempotency.
  - Owns plan state transitions (`PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`, `CANCELLED`).

- `internal/controlplane/ingest`
  - Responsibility: handle heartbeat upserts and execution status ingestion.
  - Must be safe for duplicate/out-of-order heartbeats.

- `internal/controlplane/netbird`
  - Responsibility: SaaS-side calls to NetBird API only.
  - No runtime dependencies from other modules except typed interface.

- `internal/agent/runtime/cloudhypervisor`
  - Responsibility: local microVM lifecycle implementation via `cloud-hypervisor` process management and config generation.
  - API surface (for agent executor): `CreateVM`, `StartVM`, `StopVM`, `DeleteVM`, `InspectVM`.

- `internal/agent/network/netbird`
  - Responsibility: join/check NetBird peer state via local CLI/API integration.
  - API surface: `JoinPeer`, `PeerStatus`, `ConnectivityCheck`.

- `internal/agent/executor`
  - Responsibility: execute plan operations atomically per VM and emit logs/status.
  - Depends only on runtime/network interfaces, not concrete implementations.

- `internal/agent/store`
  - Responsibility: local persistence for dedupe and crash recovery.
  - Backing store for MVP-1: embedded SQLite file `/var/lib/nkudo/agent.db`.

