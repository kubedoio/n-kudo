# n-kudo MVP-1 Architecture (SaaS control-plane + single edge agent)

## Scope and non-goals
- Scope: onboard one customer site ("Sub-cloud") by installing one agent service on an existing Linux host, enroll it securely, join NetBird, collect host inventory, and manage Cloud Hypervisor microVM lifecycle (create/start/stop/delete).
- Non-goals for MVP-1: multi-host site orchestration, advanced network load balancing, autoscaling, onsite control-plane components, and non-Linux edge support.

## Component diagram (text)

### SaaS control-plane (EU region)
- `api-gateway`: public HTTPS endpoint for dashboard/API clients.
- `auth-service`: user authN/authZ, tenant-scoped RBAC, JWT issuance.
- `enrollment-service`: validates enrollment tokens, issues site/agent identities.
- `agent-ingest-service`: gRPC endpoint for heartbeat, host facts, execution status, and logs.
- `plan-service`: persists desired actions (ApplyPlan), tracks execution state.
- `netbird-adapter`: integrates with NetBird management API for peer registration metadata and connectivity checks.
- `audit-service`: immutable audit event write path for security/compliance actions.
- `postgres`: primary relational store for tenancy, inventory, plans, executions, and audit metadata.
- `log-store`: append-only execution log storage (Postgres table in MVP-1, object storage later).

### Edge data-plane (customer host)
- `nkudo-agent` (single static binary, one systemd unit):
  - enrollment client
  - mTLS/gRPC control channel client
  - host facts collector (CPU, memory, storage, kernel, virtualization support)
  - NetBird module (join/check connectivity)
  - Cloud Hypervisor provider (create/start/stop/delete microVM)
  - plan executor with local idempotency cache
  - log forwarder
- `netbird` daemon (existing package requirement on host) used by the agent module.
- `cloud-hypervisor` binary used by the provider module.

## Data flows

### 1) Enrollment flow
1. Tenant admin creates a one-time `enrollment_token` in dashboard (TTL 15 minutes, single use, site-scoped).
2. Installer runs agent with token: `nkudo-agent enroll --token <token> --control-plane <fqdn>`.
3. Agent opens TLS connection to `enrollment-service` and sends token + host fingerprint.
4. Service validates token and tenant/site binding, writes `agents` row, issues:
   - `agent_id`
   - short-lived bootstrap certificate (or signed CSR response)
   - refresh credential bound to agent + site
5. Agent stores credentials under `/var/lib/nkudo/` (0600 root), starts systemd service.
6. Enrollment event is written to `audit_events`.

### 2) Heartbeat + HostFacts flow
1. Agent sends heartbeat every 15 seconds (jittered) via gRPC mTLS.
2. Payload includes:
   - liveness and software version
   - host capacity/allocations
   - NetBird peer status
   - microVM summaries
   - execution progress updates
3. Control-plane validates mTLS identity, enforces tenant/site ownership, upserts host + agent state.
4. Response includes pending plans (if any) and heartbeat interval override.
5. If no pending plans, agent continues loop.

### 3) ApplyPlan execution flow
1. Dashboard/API calls `ApplyPlan` (tenant-scoped) with exact operations (create/start/stop/delete microVM).
2. `plan-service` stores immutable plan + generated `plan_id` and `plan_version`.
3. Agent receives plan in next heartbeat response.
4. Agent executes step-by-step using Cloud Hypervisor module, each step idempotent by `operation_id`.
5. Agent streams execution logs and final status.
6. Control-plane updates `executions` and `microvms` states, surfaces to dashboard.
7. Every transition emits `audit_events`.

## Threat model summary

Top risks and MVP-1 mitigations:
- Stolen enrollment token -> single-use + short TTL + optional IP allow-list + audit trail + manual revoke.
- Agent impersonation -> mTLS with certs issued only after token validation, cert rotation every 24h, cert revocation list check on ingest.
- Cross-tenant data leakage -> every row and query scoped by `tenant_id`, auth middleware injects tenant context, explicit compound unique keys (`tenant_id`, resource_id).
- Plan replay/duplicate execution -> `idempotency_key` on ApplyPlan, monotonic `plan_version`, per-operation dedupe key persisted by agent.
- Command tampering in transit -> TLS 1.3, signed plan payload hash stored with execution record.
- Host compromise leaking secrets -> least-privilege filesystem perms, no long-term plaintext tokens, support credential revoke from control-plane.
- Excessive telemetry / GDPR breach -> collect only operational metadata (no guest payloads), EU data residency, configurable log retention (30 days default), delete workflows for tenant offboarding.

## Key decisions

### gRPC vs HTTPS
- Agent <-> control-plane: gRPC over HTTP/2 with mTLS.
  - Reason: efficient bidirectional streaming for logs, compact schema contracts, stricter typed protocol.
- Dashboard/API client <-> control-plane: HTTPS/JSON (OpenAPI can be generated from internal services).
  - Reason: browser/client compatibility and lower integration friction.

### Certificates
- SaaS-operated private CA issues short-lived client certs to agents.
- Agent renews cert before expiry through authenticated refresh call.
- Server certs from standard public CA; mTLS required on ingest endpoints.

### Token enrollment
- One-time enrollment token generated by tenant admin.
- Token claims: `tenant_id`, `site_id`, `expires_at`, `nonce`, `max_uses=1`.
- Token is never stored plaintext; store `token_hash` and compare constant-time.

### Idempotency
- `ApplyPlanRequest.idempotency_key` required and unique per tenant for 24h.
- Agent tracks `operation_id` execution outcomes locally and reports them upstream.
- Control-plane deduplicates retries and returns previously created `plan_id` when key matches.

