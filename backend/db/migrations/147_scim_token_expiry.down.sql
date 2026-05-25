DROP INDEX IF EXISTS idx_scim_tokens_expires_at;
ALTER TABLE scim_tokens DROP COLUMN IF EXISTS expires_at;
