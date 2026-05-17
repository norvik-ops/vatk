-- Reverse of Migration 090.

DROP INDEX IF EXISTS idx_ck_evidence_expiry_unnotified;

ALTER TABLE ck_evidence DROP COLUMN IF EXISTS expiry_notified_at;
