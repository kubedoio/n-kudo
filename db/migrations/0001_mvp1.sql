BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE agent_state AS ENUM ('ONLINE', 'DEGRADED', 'OFFLINE');
CREATE TYPE plan_status AS ENUM ('PENDING', 'IN_PROGRESS', 'SUCCEEDED', 'FAILED', 'CANCELLED');
CREATE TYPE execution_state AS ENUM ('PENDING', 'IN_PROGRESS', 'SUCCEEDED', 'FAILED');
CREATE TYPE microvm_state AS ENUM ('CREATING', 'STOPPED', 'RUNNING', 'DELETING', 'ERROR');

CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  primary_region TEXT NOT NULL DEFAULT 'eu-central-1',
  data_retention_days INTEGER NOT NULL DEFAULT 30 CHECK (data_retention_days >= 7),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('OWNER', 'ADMIN', 'OPERATOR', 'VIEWER')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email),
  UNIQUE (id, tenant_id)
);

CREATE TABLE sites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  external_key TEXT,
  location_country_code CHAR(2),
  netbird_network_id TEXT,
  connectivity_state TEXT NOT NULL DEFAULT 'OFFLINE' CHECK (connectivity_state IN ('ONLINE', 'PARTIAL', 'OFFLINE')),
  last_heartbeat_at TIMESTAMPTZ,
  created_by_user_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name),
  UNIQUE (tenant_id, external_key),
  UNIQUE (id, tenant_id)
);

ALTER TABLE sites
  ADD CONSTRAINT fk_sites_created_by_user
  FOREIGN KEY (created_by_user_id, tenant_id) REFERENCES users(id, tenant_id);

CREATE TABLE hosts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  hostname TEXT NOT NULL,
  cpu_cores_total INTEGER NOT NULL DEFAULT 0,
  memory_bytes_total BIGINT NOT NULL DEFAULT 0,
  storage_bytes_total BIGINT NOT NULL DEFAULT 0,
  kvm_available BOOLEAN NOT NULL DEFAULT FALSE,
  cloud_hypervisor_available BOOLEAN NOT NULL DEFAULT FALSE,
  last_facts_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, site_id, hostname),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE agents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  host_id UUID NOT NULL,
  enrollment_token_hash TEXT,
  refresh_token_hash TEXT NOT NULL,
  cert_serial TEXT,
  agent_version TEXT NOT NULL,
  os TEXT NOT NULL,
  arch TEXT NOT NULL,
  kernel_version TEXT,
  state agent_state NOT NULL DEFAULT 'OFFLINE',
  heartbeat_seq BIGINT NOT NULL DEFAULT 0,
  enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_heartbeat_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (id, tenant_id),
  UNIQUE (tenant_id, site_id, host_id),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (host_id, tenant_id) REFERENCES hosts(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE microvms (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  host_id UUID NOT NULL,
  name TEXT NOT NULL,
  state microvm_state NOT NULL DEFAULT 'CREATING',
  vcpu_count INTEGER NOT NULL,
  memory_mib BIGINT NOT NULL,
  kernel_image_path TEXT,
  rootfs_path TEXT,
  net_ifaces JSONB NOT NULL DEFAULT '[]'::jsonb,
  cloud_hypervisor_vm_ref TEXT,
  last_transition_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, site_id, name),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (host_id, tenant_id) REFERENCES hosts(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  requested_by_user_id UUID,
  idempotency_key TEXT NOT NULL,
  client_request_id TEXT,
  plan_version BIGINT NOT NULL,
  status plan_status NOT NULL DEFAULT 'PENDING',
  operations_json JSONB NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key),
  UNIQUE (site_id, plan_version),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE
);

ALTER TABLE plans
  ADD CONSTRAINT fk_plans_requested_by_user
  FOREIGN KEY (requested_by_user_id, tenant_id) REFERENCES users(id, tenant_id);

CREATE TABLE executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID NOT NULL,
  host_id UUID,
  agent_id UUID,
  plan_id UUID NOT NULL,
  vm_id UUID,
  operation_id TEXT NOT NULL,
  operation_type TEXT NOT NULL CHECK (operation_type IN ('CREATE', 'START', 'STOP', 'DELETE')),
  state execution_state NOT NULL DEFAULT 'PENDING',
  error_code TEXT,
  error_message TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (plan_id, operation_id),
  UNIQUE (id, tenant_id),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (host_id, tenant_id) REFERENCES hosts(id, tenant_id),
  FOREIGN KEY (agent_id, tenant_id) REFERENCES agents(id, tenant_id),
  FOREIGN KEY (plan_id, tenant_id) REFERENCES plans(id, tenant_id) ON DELETE CASCADE,
  FOREIGN KEY (vm_id, tenant_id) REFERENCES microvms(id, tenant_id)
);

CREATE TABLE execution_logs (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  execution_id UUID NOT NULL,
  sequence BIGINT NOT NULL,
  severity TEXT NOT NULL CHECK (severity IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
  message TEXT NOT NULL,
  emitted_at TIMESTAMPTZ NOT NULL,
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, execution_id, sequence),
  FOREIGN KEY (execution_id, tenant_id) REFERENCES executions(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  site_id UUID,
  actor_type TEXT NOT NULL CHECK (actor_type IN ('USER', 'AGENT', 'SYSTEM')),
  actor_user_id UUID,
  actor_agent_id UUID,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  request_id TEXT,
  source_ip INET,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (site_id, tenant_id) REFERENCES sites(id, tenant_id),
  FOREIGN KEY (actor_user_id, tenant_id) REFERENCES users(id, tenant_id),
  FOREIGN KEY (actor_agent_id, tenant_id) REFERENCES agents(id, tenant_id)
);

CREATE INDEX idx_sites_tenant_last_heartbeat
  ON sites (tenant_id, last_heartbeat_at DESC);

CREATE INDEX idx_hosts_site
  ON hosts (tenant_id, site_id);

CREATE INDEX idx_agents_site_state
  ON agents (tenant_id, site_id, state);

CREATE INDEX idx_microvms_site_state
  ON microvms (tenant_id, site_id, state);

CREATE INDEX idx_plans_site_status
  ON plans (tenant_id, site_id, status, created_at DESC);

CREATE INDEX idx_executions_plan_state
  ON executions (tenant_id, plan_id, state, updated_at DESC);

CREATE INDEX idx_execution_logs_execution_time
  ON execution_logs (tenant_id, execution_id, emitted_at DESC);

CREATE INDEX idx_audit_events_tenant_time
  ON audit_events (tenant_id, occurred_at DESC);

COMMIT;
