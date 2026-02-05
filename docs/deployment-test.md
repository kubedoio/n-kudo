# Deployment Test Runbook

This runbook validates MVP-1 deployment behavior on a single Linux host.

## Scope

- Control-plane starts via `docker compose`
- Edge enrolls and sends heartbeat with host facts
- Plan is submitted and executed for microVM lifecycle
- Status, logs, and VM state are queryable
- Cleanup (stop/delete) completes

## Prerequisites

Required:

- Linux host
- Docker + Docker Compose (`docker compose`)
- Go (matching project toolchain)
- `curl`, `jq`, `openssl`

Required for full microVM lifecycle test:

- Root privileges (`sudo`)
- `cloud-hypervisor` in `PATH`
- `ip` command (`iproute2`)
- `cloud-localds` or `genisoimage`
- `/dev/kvm` access if running with hardware acceleration

## 1. Clean Start

```bash
docker compose -f deployments/docker-compose.yml down -v || true
sudo rm -rf .demo/mvp1
```

## 2. Build Binaries

```bash
go build -o bin/control-plane ./cmd/control-plane
go build -o bin/edge ./cmd/edge
```

## 3. Baseline Validation

```bash
go test ./...
go vet ./...
```

## 4. Run Deployment Test

```bash
sudo -E ./demo.sh
```

The script performs:

- `docker compose up -d --build postgres backend`
- Tenant/site bootstrap
- Enrollment token issuance
- `n-kudo-edge enroll`
- Initial heartbeat with host facts
- Plan submit (`CREATE`, `START`) and local execution
- Status/log/VM-state queries
- Plan submit (`STOP`, `DELETE`) and local execution
- Final VM cleanup verification

## 5. Pass Criteria

Treat the deployment test as passing when all are true:

- `demo.sh` exits with code `0`
- Output includes `Demo complete.`
- Output includes `local runtime cleanup confirmed`
- `/healthz` returns `200`
- At least one host appears from:
  - `GET /sites/{siteID}/hosts`
- VM records are visible during lifecycle from:
  - `GET /sites/{siteID}/vms`
- Execution logs are returned from:
  - `GET /executions/{executionID}/logs`

## 6. Teardown

```bash
docker compose -f deployments/docker-compose.yml down
```

To remove volumes and all local demo state:

```bash
docker compose -f deployments/docker-compose.yml down -v
sudo rm -rf .demo/mvp1
```

## 7. Troubleshooting

Use:

- `docs/mvp1/demo-troubleshooting.md`

Quick checks:

```bash
docker compose -f deployments/docker-compose.yml ps
docker compose -f deployments/docker-compose.yml logs backend
docker compose -f deployments/docker-compose.yml logs postgres
```
