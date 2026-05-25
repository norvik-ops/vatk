-- Migration 147: Add expires_at to scim_tokens for automatic token expiration.
-- Tokens without expires_at never expire (preserves existing behaviour).
-- The auto-revocation Asynq job (TaskSCIMTokenExpiry) sets revoked_at = NOW()
-- for rows where expires_at IS NOT NULL AND expires_at < NOW().
ALTER TABLE scim_tokens
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_scim_tokens_expires_at
    ON scim_tokens (expires_at)
    WHERE expires_at IS NOT NULL AND revoked_at IS NULL;
