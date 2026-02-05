# Repository Structure (Canonical)

This document is the source of truth for repository layout.

## Target Layout

```text
n-kudo/
  cmd/
    control-plane/
    edge/
    worker/                # optional
  internal/
    controlplane/
      api/
      auth/
      db/
      enroll/
      plans/
      audit/
    edge/
      enroll/
      mtls/
      hostfacts/
      executor/
      state/
      netbird/
      providers/
        cloudhypervisor/
    shared/
      model/
      crypto/
      proto/               # only when proto/grpc artifacts are used
  deployments/
    docker-compose.yml
  scripts/
    install-edge.sh
    dev-up.sh
  api/
    openapi.yaml
  docs/
    architecture.md
    repo-structure.md
```

## Boundaries

- `internal/controlplane/...` must not import `internal/edge/...`.
- `internal/edge/...` must not import `internal/controlplane/...`.
- Shared contracts/helpers belong in `internal/shared/...` only if truly cross-domain.

## Placement Rules

- Entrypoints are only in `cmd/<binary>/main.go`.
- Runtime app code belongs in `internal/...`.
- Do not reintroduce old runtime paths like top-level `pkg/` or nested runtime modules.
- Keep public package surface minimal.

## Validation Checklist

Run after structural changes:

```bash
go test ./...
go vet ./...
go build -o bin/control-plane ./cmd/control-plane
go build -o bin/edge ./cmd/edge
```
