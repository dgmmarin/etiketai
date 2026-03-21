CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS workspaces (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(255) NOT NULL,
    cui                     VARCHAR(20),
    address                 TEXT,
    phone                   VARCHAR(50),
    plan                    VARCHAR(50) NOT NULL DEFAULT 'starter',
    label_quota_monthly     INTEGER NOT NULL DEFAULT 100,
    label_quota_used        INTEGER NOT NULL DEFAULT 0,
    stripe_customer_id      VARCHAR(100),
    stripe_subscription_id  VARCHAR(100),
    subscription_expires_at TIMESTAMPTZ,
    logo_s3_key             VARCHAR(500),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_stripe_customer ON workspaces (stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS workspace_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,   -- cross-service reference (auth_db)
    email               VARCHAR(255),    -- denormalized for display
    role                VARCHAR(50) NOT NULL DEFAULT 'operator',
    invite_token_hash   VARCHAR(255),
    invited_by_user_id  UUID,
    invited_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at         TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    UNIQUE (workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_workspace_id ON workspace_members (workspace_id);
CREATE INDEX idx_workspace_members_user_id      ON workspace_members (user_id);
CREATE INDEX idx_workspace_members_role         ON workspace_members (workspace_id, role);
CREATE UNIQUE INDEX idx_workspace_members_token ON workspace_members (invite_token_hash) WHERE invite_token_hash IS NOT NULL;
