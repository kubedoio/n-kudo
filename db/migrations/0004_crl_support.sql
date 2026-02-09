BEGIN;

-- CRL entries table for certificate revocation
CREATE TABLE IF NOT EXISTS crl_entries (
    serial TEXT PRIMARY KEY,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason INTEGER NOT NULL DEFAULT 0,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL
);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_crl_entries_revoked_at 
    ON crl_entries (revoked_at);

CREATE INDEX IF NOT EXISTS idx_crl_entries_agent_id 
    ON crl_entries (agent_id) WHERE agent_id IS NOT NULL;

COMMIT;
