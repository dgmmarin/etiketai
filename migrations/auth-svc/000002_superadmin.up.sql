-- Add platform superadmin flag to users.
-- Idempotent: safe to run multiple times.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_superadmin BOOLEAN NOT NULL DEFAULT false;
