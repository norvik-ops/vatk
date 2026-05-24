ALTER TABLE organizations DROP COLUMN IF EXISTS ai_weekly_digest_enabled;
ALTER TABLE ck_risks DROP COLUMN IF EXISTS ai_narrative;
DROP INDEX IF EXISTS idx_ck_ai_insights_org_active;
DROP TABLE IF EXISTS ck_ai_insights;
