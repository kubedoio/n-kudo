# n-kudo

Monorepo layout:

- `cmd/control-plane` - control-plane binary
- `cmd/edge` - edge agent binary
- `internal/controlplane` - control-plane internals
- `internal/edge` - edge internals

Quick start:

```bash
make test
make build-cp
make build-edge
```
