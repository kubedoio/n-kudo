# MVP-1 Repo Layout and Module Boundaries

## Current monorepo structure

```text
n-kudo/
  api/
    proto/controlplane/v1/controlplane.proto
    openapi/
  cmd/
    control-plane/main.go
    edge/main.go
  internal/
    controlplane/
      api/                  # API handlers + request validation
      auth/                 # authN/authZ middleware
      db/                   # persistence adapters + migrations wiring
      enroll/               # enrollment token and cert bootstrap logic
      plans/                # plan creation and lifecycle state
      audit/                # audit event write path
    edge/
      enroll/               # enrollment client + identity persistence
      mtls/                 # key/cert generation + TLS client setup
      hostfacts/            # host inventory collection
      executor/             # plan/action execution + idempotency behavior
      state/                # local state persistence
      netbird/              # NetBird join/status/probe integration
      providers/
        cloudhypervisor/    # Cloud Hypervisor runtime implementation
  deployments/
    docker/
    systemd/nkudo-edge.service
  scripts/
    install-edge.sh
    dev-up.sh
  tests/
    integration/
  docs/
    repo-structure.md
    mvp1/
```

## Module boundaries

- `internal/controlplane/api`
  - Responsibility: public API surface for dashboard/agent interactions.
  - Must enforce tenant scoping and auth before domain logic.

- `internal/controlplane/plans`
  - Responsibility: immutable plan records, idempotency (`idempotency_key`), and state transitions.

- `internal/controlplane/enroll`
  - Responsibility: one-time token validation and enrollment response generation.

- `internal/edge/enroll`
  - Responsibility: token/CSR enrollment call and secure local identity write.

- `internal/edge/mtls`
  - Responsibility: cert/key generation, secure PKI file permissions, mTLS client config.

- `internal/edge/hostfacts`
  - Responsibility: CPU/RAM/disk/OS/kernel/KVM/interface/bridge facts collection.

- `internal/edge/netbird`
  - Responsibility: local NetBird status/join/probe checks only.

- `internal/edge/providers/cloudhypervisor`
  - Responsibility: microVM create/start/stop/delete operations.

- `internal/edge/executor`
  - Responsibility: execute plan actions in order with idempotent action replay handling.

- `internal/edge/state`
  - Responsibility: local state and execution metadata persistence used by executor/provider.

## Placement rules

- Entrypoints live only in `cmd/control-plane` and `cmd/edge`.
- Runtime business logic lives under `internal/...`.
- Avoid reintroducing top-level `pkg/` runtime modules.
