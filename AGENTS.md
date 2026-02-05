# Agent Instructions For This Repository

This repository is a Go monorepo with a fixed structure. Do not invent alternate layouts.

## Canonical Structure

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
      proto/               # only if gRPC/proto artifacts are used
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

## Hard Rules

1. Keep the module path as `github.com/kubedoio/n-kudo`.
2. Keep binaries in `cmd/<name>/main.go`.
3. Do not create or re-introduce a nested edge module under `/edge` for runtime code.
4. Do not place runtime packages under `/pkg`; use `/internal/...`.
5. No cross-imports between:
   - `internal/edge/...` and `internal/controlplane/...`
   - shared code must live in `internal/shared/...` and stay minimal.
6. Prefer `git mv` for moves when possible.
7. After refactors, run:
   - `go test ./...`
   - `go vet ./...`
   - `go build -o bin/control-plane ./cmd/control-plane`
   - `go build -o bin/edge ./cmd/edge`

## Source Of Truth

For folder layout decisions, use `/Users/scolak/Projects/n-kudo/docs/repo-structure.md`.
