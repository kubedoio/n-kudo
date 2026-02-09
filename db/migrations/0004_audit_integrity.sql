BEGIN;

-- Add chain integrity columns to audit_events table
ALTER TABLE audit_events
  ADD COLUMN IF NOT EXISTS prev_hash TEXT NOT NULL DEFAULT '0000000000000000000000000000000000000000000000000000000000000000',
  ADD COLUMN IF NOT EXISTS entry_hash TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS chain_valid BOOLEAN NOT NULL DEFAULT TRUE;

-- Create index for chain validation queries
CREATE INDEX IF NOT EXISTS idx_audit_events_chain
  ON audit_events (id, entry_hash, chain_valid);

-- Create index for tenant-scoped audit queries with chain status
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_chain
  ON audit_events (tenant_id, occurred_at DESC, chain_valid);

COMMIT;
