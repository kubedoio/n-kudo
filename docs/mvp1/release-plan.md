# MVP-1 Build and Release Plan

## Build artifacts

### Edge agent (static binary)
- Target: `nkudo-edge` static Linux binary.
- Build flags:
  - `CGO_ENABLED=0`
  - `GOOS=linux`
  - `GOARCH=amd64` and `arm64`
- Suggested command:
  - `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o dist/nkudo-edge-linux-amd64 ./cmd/edge`
- Distribution:
  - tarball with `nkudo-edge`, sample config, and `systemd` unit.
  - sha256 checksum + detached signature.

### Control-plane
- Build container image from `cmd/control-plane`.
- Tag with immutable version and git SHA.

## Versioning
- Use SemVer:
  - `MAJOR.MINOR.PATCH` for both control-plane and agent.
- Compatibility rule:
  - control-plane supports current and previous minor agent versions (N and N-1).
- Record in heartbeat:
  - `agent_version` and `minimum_supported_agent_version` check response.

## CI/CD pipeline
1. Lint + unit tests.
2. Protobuf/code generation check.
3. Integration tests with mock NetBird and mock Cloud Hypervisor.
4. Build artifacts (agent binaries, control-plane image).
5. Sign artifacts and publish release manifest.
6. Deploy control-plane to staging, run smoke test.
7. Promote to production (EU region).

## Rollout strategy

### Control-plane
- Canary 10% traffic for ingest and ApplyPlan APIs.
- Monitor:
  - enroll success rate
  - heartbeat ingest latency (p95)
  - plan execution failure rate
- Promote to 100% after 30-60 min healthy metrics.

### Agent
- Rollout waves by tenant/site:
  - Wave 1: internal test tenant
  - Wave 2: 2 design partners
  - Wave 3: all MVP customers
- Upgrade mechanism:
  - dashboard surfaces download URL; customer updates binary + restarts systemd service.

## Rollback strategy
- Control-plane:
  - keep previous image available; rollback via deployment tool to prior stable revision.
  - DB migrations must be backward-compatible for one minor version.
- Agent:
  - keep previous binary in `/opt/nkudo/releases/<version>/`.
  - rollback command: symlink previous binary and `systemctl restart nkudo-edge`.
- Trigger conditions:
  - >5% plan failure increase
  - enrollment failure >2% absolute increase
  - repeated agent crash loop on upgrade

## Operational readiness checklist
- Runbook for enrollment failure, netbird join failure, and Cloud Hypervisor command failure.
- Alerting configured for ingest downtime and tenant isolation violations.
- Audit log verification in production before enabling customer onboarding.
