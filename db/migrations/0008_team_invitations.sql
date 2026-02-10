BEGIN;

-- Project invitations table
CREATE TABLE project_invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('ADMIN', 'OPERATOR', 'VIEWER')),
  invited_by_user_id UUID NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'DECLINED', 'CANCELLED', 'EXPIRED')),
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  declined_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email, status) WHERE status = 'PENDING'
);

CREATE INDEX idx_project_invitations_tenant ON project_invitations (tenant_id, status);
CREATE INDEX idx_project_invitations_email ON project_invitations (email, status);
CREATE INDEX idx_project_invitations_token ON project_invitations (token_hash);
CREATE INDEX idx_project_invitations_expires ON project_invitations (expires_at) WHERE status = 'PENDING';

-- Add invited_by tracking to users table (optional, for audit)
-- Note: users are already linked to tenant_id

-- Trigger to clean up expired invitations (optional - can also be done in application)
-- Or we can have a background job

COMMIT;
