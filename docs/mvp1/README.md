# MVP-1 Deliverables Index

- Architecture doc: `docs/mvp1/architecture.md`
- Repo layout and module boundaries: `docs/mvp1/repo-layout.md`
- API contracts (protobuf): `api/proto/controlplane/v1/controlplane.proto`
- Postgres schema: `db/migrations/0001_mvp1.sql`
- Acceptance criteria and test plan: `docs/mvp1/acceptance-and-test-plan.md`
- Build/release plan: `docs/mvp1/release-plan.md`
- 4-agent task breakdown: `docs/mvp1/task-breakdown.md`
- NetBird MVP-1 strategy + runbook: `docs/mvp1/netbird-mvp1.md`

Notes:
- API and DB field names are intentionally aligned for direct mapping.
- Schema includes required MVP-1 tables plus `execution_logs` for the log-streaming requirement.
