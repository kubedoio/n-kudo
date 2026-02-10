BEGIN;

-- Add password and auth fields to users table
ALTER TABLE users
  ADD COLUMN password_hash TEXT NOT NULL DEFAULT '',
  ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN last_login_at TIMESTAMPTZ,
  ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Remove the default after adding (we'll require proper passwords)
ALTER TABLE users ALTER COLUMN password_hash DROP DEFAULT;

-- Create index for email lookups during login
CREATE INDEX idx_users_email ON users (email) WHERE is_active = TRUE;

-- Create password reset tokens table
CREATE TABLE password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (token_hash)
);

CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens (user_id, tenant_id);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens (expires_at) WHERE used_at IS NULL;

COMMIT;
