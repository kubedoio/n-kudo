# n-kudo

Monorepo layout:

- `cmd/control-plane` - control-plane binary
- `cmd/edge` - edge agent binary
- `internal/controlplane` - control-plane internals
- `internal/edge` - edge internals
- Canonical structure rules: `/Users/scolak/Projects/n-kudo/docs/repo-structure.md`
- Agent guardrails: `/Users/scolak/Projects/n-kudo/AGENTS.md`
- Deployment test runbook: `/Users/scolak/Projects/n-kudo/docs/deployment-test.md`

Quick start:

```bash
make test
make build-cp
make build-edge
```
