# n-kudo (MVP-1)

[![Go Version](https://img.shields.io/badge/go-1.23-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

A SaaS control plane for managing edge computing resources with secure agent enrollment, microVM lifecycle management, and mTLS-secured communication.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SaaS Control Plane                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   API GW     │  │   Auth       │  │  Enrollment  │  │   Agent      │     │
│  │   (HTTPS)    │──│   Service    │──│   Service    │──│   Ingest     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│         │                 │                  │                  │           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Plan Service │  │ NetBird      │  │   Audit      │  │  PostgreSQL  │     │
│  │              │  │   Adapter    │  │   Service    │  │   (State)    │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                        mTLS + HTTP/2 (Agent Channel)
                                    │
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Edge Data-Plane (Per Site)                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        nkudo-edge Agent                            │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │   │
│  │  │  Enroll  │  │  mTLS    │  │  Host    │  │    Cloud         │   │   │
│  │  │  Client  │──│  Client  │──│  Facts   │──│  Hypervisor      │   │   │
│  │  └──────────┘  └──────────┘  └──────────┘  │  Provider        │   │   │
│  │                                             │  (MicroVMs)      │   │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  └──────────────────┘   │   │
│  │  │  Plan    │  │  State   │  │ NetBird  │                        │   │
│  │  │ Executor │  │  Store   │  │  Module  │                        │   │
│  │  └──────────┘  └──────────┘  └──────────┘                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Description |
|-----------|-------------|
| **Control Plane** | Centralized management API with tenant isolation, enrollment, and plan orchestration |
| **Edge Agent** | Single binary deployed on edge hosts for local execution and state management |
| **mTLS** | Mutual TLS for all agent-to-control-plane communication with short-lived certificates |
| **Cloud Hypervisor** | Lightweight VMM for microVM lifecycle (create/start/stop/delete) |
| **NetBird** | Optional VPN integration for secure mesh networking between sites |

## Snapshot (Current State)

Validated on **February 6, 2026** in this repo:

- `go test ./...` passes
- `go vet ./...` passes
- `go build -o bin/control-plane ./cmd/control-plane` passes
- `go build -o bin/edge ./cmd/edge` passes

## For LLM Agents: Read This First

Before editing anything, read in this order:

1. `AGENTS.md` (hard guardrails and repository rules)
2. `docs/repo-structure.md` (canonical layout source of truth)
3. `docs/mvp1/README.md` (MVP-1 deliverables index)
4. `docs/mvp1/architecture.md` (target architecture and flows)

Then verify repo state:

```bash
git status --short
go test ./...
go vet ./...
go build -o bin/control-plane ./cmd/control-plane
go build -o bin/edge ./cmd/edge
```

## Canonical Repo Rules

- Go module path must remain `github.com/kubedoio/n-kudo`
- Runtime code belongs in `internal/...`, not `pkg/...`
- Entrypoints must be in `cmd/<name>/main.go`
- No cross-imports between `internal/controlplane/...` and `internal/edge/...`
- Shared code should remain minimal in `internal/shared/...`

Reference:
- `AGENTS.md`
- `docs/repo-structure.md`

## Repository Map

- `cmd/control-plane/main.go`: control-plane process (`serve`, `migrate`)
- `cmd/edge/main.go`: edge CLI (`enroll`, `run`, `hostfacts`, `apply`, `verify-heartbeat`)
- `internal/controlplane/api`: HTTP/TLS server and endpoint handlers
- `internal/controlplane/db`: repo interfaces + Postgres + in-memory store
- `internal/controlplane/db/migrate`: SQL migration runner
- `internal/edge/enroll`: enrollment + control-plane HTTP client
- `internal/edge/mtls`: key/CSR/cert helpers and mTLS clients
- `internal/edge/hostfacts`: host telemetry collection
- `internal/edge/executor`: plan/action execution + local idempotency cache usage
- `internal/edge/providers/cloudhypervisor`: VM provider implementation
- `internal/edge/netbird`: NetBird readiness checks and optional join
- `internal/edge/state`: local JSON state store (`edge-state.json`)
- `db/migrations`: Postgres schema for MVP-1
- `deployments`: Docker Compose + systemd + Dockerfile
- `demo.sh`: end-to-end local MVP-1 run script

Note:
- `edge/` exists as a separate nested module with vendored third-party dependencies; do not place new runtime code there.

## MVP-1 Implementation Coverage

Implemented now:

- Tenant bootstrap: create tenant, API key, site
- One-time enrollment tokens (issue + consume)
- Agent enrollment with signed client cert issuance
- Heartbeat ingest to persist host + microVM + execution state updates
- Plan creation/idempotency via `POST /sites/{siteID}/plans`
- Execution log ingestion (`/agents/logs` and `/v1/logs`)
- Host/VM/log query endpoints for dashboard-style polling
- Edge local action execution with per-action idempotency cache
- Cloud Hypervisor provider create/start/stop/delete flow
- NetBird status evaluation with optional auto-join and probe

Partially implemented / placeholders:

- gRPC/proto contracts are defined (`api/proto/...`) but runtime server is HTTP/JSON

## Control-Plane Commands

```bash
# Run server
go run ./cmd/control-plane serve

# Apply SQL migrations from db/migrations
go run ./cmd/control-plane migrate -dir db/migrations
```

If no subcommand is provided, control-plane defaults to `serve`.

## Control-Plane Configuration

Environment variables from `internal/controlplane/api/config.go` and `internal/controlplane/api/pki.go`:

| Variable | Default | Purpose |
|---|---|---|
| `CONTROL_PLANE_ADDR` | `:8443` | HTTPS listen address |
| `DATABASE_URL` | `postgres://nkudo:nkudo@localhost:5432/nkudo?sslmode=disable` | Postgres DSN |
| `ADMIN_KEY` | `dev-admin-key` | Admin bootstrap auth header (`X-Admin-Key`) |
| `DEFAULT_ENROLLMENT_TTL` | `15m` | Enrollment token TTL |
| `AGENT_CERT_TTL` | `24h` | Agent mTLS cert TTL |
| `HEARTBEAT_INTERVAL` | `15s` | Agent heartbeat interval override returned by control-plane |
| `PLAN_LEASE_TTL` | `45s` | Lease TTL for pending plans handed to an agent |
| `MAX_PENDING_PLANS` | `2` | Max plans returned per heartbeat or `/v1/plans/next` |
| `HEARTBEAT_OFFLINE_AFTER` | `60s` | Mark agents offline if heartbeat age exceeds this duration |
| `OFFLINE_SWEEP_INTERVAL` | `15s` | Background sweeper cadence for offline-state transitions |
| `REQUIRE_PERSISTENT_PKI` | `false` | If `true`, startup fails unless CA/server cert files are configured |
| `HTTP_READ_TIMEOUT` | `10s` | Server read timeout |
| `HTTP_WRITE_TIMEOUT` | `15s` | Server write timeout |
| `HTTP_IDLE_TIMEOUT` | `60s` | Server idle timeout |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `CA_COMMON_NAME` | `n-kudo-mvp1-agent-ca` | Generated CA subject CN |
| `CA_CERT_FILE` | unset | Existing CA certificate PEM path |
| `CA_KEY_FILE` | unset | Existing CA private key PEM path |
| `SERVER_CERT_FILE` | unset | Existing server TLS cert PEM path |
| `SERVER_KEY_FILE` | unset | Server TLS key PEM path (required if cert file is set) |

Important TLS behavior:

- If `REQUIRE_PERSISTENT_PKI=false` (default) and cert files are not set, startup generates in-memory CA/server material for dev.
- If `REQUIRE_PERSISTENT_PKI=true`, startup fails unless both `CA_CERT_FILE`+`CA_KEY_FILE` and `SERVER_CERT_FILE`+`SERVER_KEY_FILE` are set.
- For non-dev deployments, set `REQUIRE_PERSISTENT_PKI=true` and provide persistent cert material.

## HTTP API Surface (Current)

Auth model:

- `X-Admin-Key`: admin bootstrap endpoints
- `X-API-Key`: tenant-scoped endpoints
- Agent mTLS cert: `/agents/*` and `/v1/*` agent endpoints

Current routes are registered in `internal/controlplane/api/server.go`.

### Bootstrap and tenant ops

- `GET /healthz`
- `POST /tenants`
- `POST /tenants/{tenantID}/api-keys`
- `POST /tenants/{tenantID}/sites`
- `GET /tenants/{tenantID}/sites`
- `POST /tenants/{tenantID}/enrollment-tokens`

### Agent ingestion

- `POST /enroll`
- `POST /v1/enroll`
- `POST /agents/heartbeat`
- `POST /v1/heartbeat`
- `POST /agents/logs`
- `POST /v1/logs`
- `GET /v1/plans/next`
- `POST /v1/executions/result`

### Plan and status queries

- `POST /sites/{siteID}/plans`
- `GET /sites/{siteID}/hosts`
- `GET /sites/{siteID}/vms`
- `GET /executions/{executionID}/logs`

## Edge CLI Commands

```bash
go run ./cmd/edge enroll
go run ./cmd/edge run
go run ./cmd/edge hostfacts
go run ./cmd/edge apply
go run ./cmd/edge verify-heartbeat
go run ./cmd/edge version
```

Default directories in edge binary:

- state: `/var/lib/nkudo-edge/state`
- pki: `/var/lib/nkudo-edge/pki`
- runtime: `/var/lib/nkudo-edge/vms`

Enrollment token lookup order:

1. `--token`
2. env `NKUDO_ENROLL_TOKEN`
3. `--token-file`

Key runtime flags:

- `--control-plane` (required for `enroll` and `run`)
- `--state-dir`, `--pki-dir`, `--runtime-dir`
- `--cloud-hypervisor-bin`
- `--heartbeat-interval`
- `--once` (single loop for `run`)
- `--insecure-skip-verify` (dev only)

NetBird flags:

- `--netbird-enabled`
- `--netbird-auto-join`
- `--netbird-setup-key`
- `--netbird-bin`
- `--netbird-hostname`
- `--netbird-install-cmd`
- `--netbird-require-service`
- `--netbird-probe-type` (`http` or `ping`)
- `--netbird-probe-target`
- `--netbird-probe-timeout`
- `--netbird-probe-http-min`
- `--netbird-probe-http-max`

## Local State and Idempotency

Edge local state lives in:

- `<state-dir>/edge-state.json`

Stored objects:

- agent identity (`tenant_id`, `site_id`, `host_id`, `agent_id`, `refresh_token`)
- microVM runtime snapshots
- action idempotency cache keyed by `action_id`

Executor behavior:

- Action types: `MicroVMCreate`, `MicroVMStart`, `MicroVMStop`, `MicroVMDelete`
- If an `action_id` exists in local cache, result is reused without re-execution

## Cloud Hypervisor Provider Notes

Provider implementation:

- `internal/edge/providers/cloudhypervisor/api.go`
- `internal/edge/providers/cloudhypervisor/provider.go`

Runtime behavior:

- Prepares per-VM runtime dir under `<runtime-dir>/<vm-id>`
- Creates/clones VM disk
- Builds cloud-init ISO using `cloud-localds` or fallback `genisoimage`/`mkisofs`
- Creates TAP device and attaches to bridge (default bridge `br0`)
- Starts `cloud-hypervisor` and tracks PID/status
- Persists command log and VM metadata in runtime directory

Current parameter mapping caveat:

- `executor.MicroVMParams.RootfsPath` is used as provider `disk_path`
- `executor.MicroVMParams.KernelPath` is currently not used by provider command rendering

## Database Schema and Migrations

Migrations:

- `db/migrations/0001_mvp1.sql`
- `db/migrations/0002_auth_tokens_plan_actions.sql`

Core tables:

- `tenants`, `sites`, `hosts`, `agents`
- `microvms`
- `plans`, `plan_actions`, `executions`
- `execution_logs`
- `api_keys`, `enrollment_tokens`
- `audit_events`
- `schema_migrations`

Requires:

- Postgres extension `pgcrypto`

## Quick Start

### Prerequisites

- Docker 24.0+ and Docker Compose 2.20+
- 4GB RAM, 2 CPU cores
- 20GB free disk space
- Linux host for edge agent (Ubuntu 22.04+ recommended)

### 1. Start Control Plane

```bash
# Clone repository
git clone https://github.com/kubedoio/n-kudo.git
cd n-kudo

# Production deployment
cp .env.production.example .env.production
# Edit .env.production with your settings
nano .env.production

# Start services
docker compose --env-file .env.production up -d

# Or use development setup
./scripts/dev-up.sh
```

### 2. Build Edge Agent

```bash
# Build binary
make build-edge

# Or run directly
go build -o bin/edge ./cmd/edge
```

### 3. Enroll Edge Agent

```bash
# Generate enrollment token (via API)
export ADMIN_KEY="your-admin-key"
export CONTROL_PLANE="https://localhost:8443"

TOKEN=$(curl -s -X POST "${CONTROL_PLANE}/tenants/${TENANT_ID}/enrollment-tokens" \
  -H "X-Admin-Key: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"ttl_minutes": 30}' | jq -r '.token')

# Enroll agent
sudo ./bin/edge enroll \
  --control-plane ${CONTROL_PLANE} \
  --token ${TOKEN}

# Start agent
sudo ./bin/edge run --control-plane ${CONTROL_PLANE}
```

### 4. Run Demo

```bash
# Automated demo flow
sudo -E ./demo.sh
```

See [Docker Compose Deployment Guide](docs/deployment/docker-compose.md) for production deployment details.
See [Kubernetes Deployment Guide](docs/deployment/kubernetes.md) for K8s deployment.

## MVP-1 Feature Checklist

### Core Platform ✅

| Feature | Status | Description |
|---------|--------|-------------|
| Tenant Management | ✅ | Multi-tenant architecture with API key authentication |
| Site Management | ✅ | Site registration and metadata tracking |
| Agent Enrollment | ✅ | One-time enrollment tokens with mTLS identity issuance |
| Certificate Lifecycle | ✅ | Short-lived agent certificates (24h TTL) with refresh |
| Heartbeat System | ✅ | 15s heartbeats with offline detection |
| Plan Execution | ✅ | Immutable plans with idempotent action execution |
| Execution Logging | ✅ | Real-time log streaming from agents |

### Edge Computing ✅

| Feature | Status | Description |
|---------|--------|-------------|
| Cloud Hypervisor | ✅ | MicroVM create/start/stop/delete lifecycle |
| Host Facts | ✅ | CPU, memory, storage, kernel telemetry |
| Local State | ✅ | JSON-based state store with idempotency cache |
| NetBird Integration | ✅ | VPN readiness checks and auto-join |

### Security ✅

| Feature | Status | Description |
|---------|--------|-------------|
| mTLS | ✅ | Mutual TLS for all agent communication |
| PKI | ✅ | Built-in CA with certificate rotation |
| API Authentication | ✅ | Admin keys + tenant-scoped API keys |
| Audit Logging | ✅ | Immutable audit event trail |
| Rate Limiting | ✅ | Per-endpoint rate limiting |

### Operations ✅

| Feature | Status | Description |
|---------|--------|-------------|
| Health Checks | ✅ | HTTP health endpoints for all services |
| Database Migrations | ✅ | Automatic schema migrations on startup |
| Metrics | ✅ | Prometheus-compatible metrics endpoint |
| Log Rotation | ✅ | Configurable log rotation policies |

## Security Considerations

### Authentication & Authorization

- **Admin API Key**: Bootstrap authentication via `X-Admin-Key` header
- **Tenant API Keys**: Scoped authentication for tenant operations
- **Agent mTLS**: Certificate-based authentication for all agent endpoints
- **Token Security**: One-time enrollment tokens with short TTL (15m default)

### Encryption

- **In Transit**: TLS 1.3 for all API communication
- **Agent Channel**: mTLS with 24-hour certificate rotation
- **At Rest**: Database encryption via PostgreSQL (configure as needed)

### Network Security

- **Default Deny**: Internal networks between services
- **Port Exposure**: Minimal external port exposure (443/8443)
- **Firewall**: Recommend restricting agent ingress to known IPs

### Production Hardening

```bash
# Required for production
REQUIRE_PERSISTENT_PKI=true
CA_CERT_FILE=/secure/path/to/ca.crt
CA_KEY_FILE=/secure/path/to/ca.key
SERVER_CERT_FILE=/secure/path/to/server.crt
SERVER_KEY_FILE=/secure/path/to/server.key

# Enable audit logging
AUDIT_LOGGING_ENABLED=true

# Set strong secrets (min 32 characters)
ADMIN_KEY=$(openssl rand -base64 32)
POSTGRES_PASSWORD=$(openssl rand -base64 32)
```

### Known Limitations

- Default TLS material is ephemeral for development convenience
- Heartbeat/log ingestion endpoints allow unknown JSON fields for forward-compatibility
- Always use persistent PKI (`REQUIRE_PERSISTENT_PKI=true`) in production

## Documentation

### Architecture & Design
- [Repository Structure](docs/repo-structure.md) - Canonical layout and boundaries
- [MVP-1 Index](docs/mvp1/README.md) - MVP-1 deliverables index
- [Architecture Details](docs/mvp1/architecture.md) - Target architecture and data flows
- [NetBird Strategy](docs/mvp1/netbird-mvp1.md) - VPN integration approach
- [Cloud Hypervisor Provider](docs/mvp1/cloudhypervisor-provider.md) - VM provider implementation

### Deployment
- [Docker Compose Guide](docs/deployment/docker-compose.md) - Production Docker Compose deployment
- [Kubernetes Guide](docs/deployment/kubernetes.md) - K8s manifests and operations

### Operations & Troubleshooting
- [Agent Troubleshooting](docs/runbooks/agent-troubleshooting.md) - Enrollment, certs, heartbeat, VM issues
- [Enrollment Failure](docs/runbooks/enrollment-failure.md) - Enrollment-specific runbook
- [VM Lifecycle Issues](docs/runbooks/vm-lifecycle-issues.md) - Cloud Hypervisor troubleshooting
- [NetBird Troubleshooting](docs/runbooks/netbird-troubleshooting.md) - VPN connectivity issues
- [Demo Troubleshooting](docs/mvp1/demo-troubleshooting.md) - Local demo issues

### Development
- [Acceptance Test Plan](docs/mvp1/acceptance-and-test-plan.md) - Testing criteria
- [AGENTS.md](AGENTS.md) - Coding standards and repository rules
