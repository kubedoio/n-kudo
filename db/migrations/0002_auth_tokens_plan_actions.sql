BEGIN;

CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  key_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_active
  ON api_keys (tenant_id, revoked_at, expires_at);

CREATE TABLE IF NOT EXISTS enrollment_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_enrollment_tokens_tenant_site
  ON enrollment_tokens (tenant_id, site_id, expires_at DESC);

CREATE TABLE IF NOT EXISTS plan_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  plan_id UUID NOT NULL,
  operation_id TEXT NOT NULL,
  operation_type TEXT NOT NULL CHECK (operation_type IN ('CREATE', 'START', 'STOP', 'DELETE')),
  vm_id UUID,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (plan_id, operation_id),
  FOREIGN KEY (plan_id, tenant_id) REFERENCES plans(id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (vm_id, tenant_id) REFERENCES microvms(id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_actions_plan
  ON plan_actions (tenant_id, plan_id, created_at ASC);

COMMIT;
