BEGIN;

-- Certificate history tracking table
CREATE TABLE IF NOT EXISTS certificate_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  agent_id UUID NOT NULL,
  serial TEXT NOT NULL,
  issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- Index for efficient lookups by agent
CREATE INDEX IF NOT EXISTS idx_certificate_history_agent
  ON certificate_history (agent_id, issued_at DESC);

-- Index for certificate serial lookups (for revocation checks)
CREATE INDEX IF NOT EXISTS idx_certificate_history_serial
  ON certificate_history (serial);

-- Index for expiry tracking (for proactive rotation notifications)
CREATE INDEX IF NOT EXISTS idx_certificate_history_expires
  ON certificate_history (expires_at)
  WHERE revoked_at IS NULL;

COMMENT ON TABLE certificate_history IS 'Tracks certificate issuance history for agents';
COMMENT ON COLUMN certificate_history.serial IS 'Certificate serial number';
COMMENT ON COLUMN certificate_history.issued_at IS 'When the certificate was issued';
COMMENT ON COLUMN certificate_history.expires_at IS 'Certificate expiration time';
COMMENT ON COLUMN certificate_history.revoked_at IS 'When the certificate was revoked (if applicable)';

COMMIT;
