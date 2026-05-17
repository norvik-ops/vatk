-- Migration 090: Add expiry notification tracking to evidence items.
-- expires_at already exists (Migration 006). This adds the dedup guard
-- so the daily worker does not send repeat notifications.

ALTER TABLE ck_evidence ADD COLUMN IF NOT EXISTS expiry_notified_at TIMESTAMPTZ;

-- Partial index for fast lookup of unnotified expiring evidence.
CREATE INDEX IF NOT EXISTS idx_ck_evidence_expiry_unnotified
  ON ck_evidence (org_id, expires_at)
  WHERE expires_at IS NOT NULL AND expiry_notified_at IS NULL;
