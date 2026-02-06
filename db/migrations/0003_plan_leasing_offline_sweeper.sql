BEGIN;

ALTER TABLE microvms
  ALTER COLUMN host_id DROP NOT NULL;

ALTER TABLE plans
  ADD COLUMN IF NOT EXISTS leased_by_agent_id UUID,
  ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;

ALTER TABLE plans
  ADD CONSTRAINT fk_plans_leased_by_agent
  FOREIGN KEY (leased_by_agent_id, tenant_id) REFERENCES agents(id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_plans_site_lease
  ON plans (tenant_id, site_id, status, lease_expires_at, created_at);

CREATE INDEX IF NOT EXISTS idx_agents_last_heartbeat
  ON agents (tenant_id, site_id, last_heartbeat_at);

COMMIT;
