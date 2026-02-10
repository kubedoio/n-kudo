BEGIN;

-- Email verification tokens
CREATE TABLE email_verification_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (token_hash)
);

CREATE INDEX idx_email_verification_tokens_user ON email_verification_tokens (user_id, tenant_id);
CREATE INDEX idx_email_verification_tokens_expires ON email_verification_tokens (expires_at) WHERE used_at IS NULL;

-- Add email_verified_at column for tracking when email was verified
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

COMMIT;
