-- Pending invites have no user_id yet; make the column nullable.
ALTER TABLE workspace_members ALTER COLUMN user_id DROP NOT NULL;

-- The old named unique constraint on (workspace_id, user_id) doesn't work
-- with NULLs (NULL != NULL in Postgres unique indexes).
-- Replace it with a partial index that only enforces uniqueness for
-- accepted members (where user_id IS NOT NULL).
ALTER TABLE workspace_members DROP CONSTRAINT IF EXISTS workspace_members_workspace_id_user_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_members_user
    ON workspace_members (workspace_id, user_id)
    WHERE user_id IS NOT NULL;

-- Prevent duplicate pending invites to the same email in the same workspace.
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_members_pending_email
    ON workspace_members (workspace_id, email)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;
