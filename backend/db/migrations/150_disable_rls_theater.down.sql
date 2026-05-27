-- Rolls Migration 150 back: re-enable RLS + recreate the original Migration
-- 012 policies. NOTE: this restores the theater (App still does not set
-- `app.current_org_id`, so the policies remain unreachable except for
-- DB-admin-side SELECT without BYPASSRLS).

ALTER TABLE vb_assets     ENABLE ROW LEVEL SECURITY;
ALTER TABLE vb_findings   ENABLE ROW LEVEL SECURITY;
ALTER TABLE ck_frameworks ENABLE ROW LEVEL SECURITY;
ALTER TABLE ck_controls   ENABLE ROW LEVEL SECURITY;
ALTER TABLE ck_evidence   ENABLE ROW LEVEL SECURITY;
ALTER TABLE so_projects   ENABLE ROW LEVEL SECURITY;
ALTER TABLE so_secrets    ENABLE ROW LEVEL SECURITY;
ALTER TABLE pg_campaigns  ENABLE ROW LEVEL SECURITY;

CREATE POLICY vb_assets_org     ON vb_assets     USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY vb_findings_org   ON vb_findings   USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY ck_frameworks_org ON ck_frameworks USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY ck_controls_org   ON ck_controls   USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY ck_evidence_org   ON ck_evidence   USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY so_projects_org   ON so_projects   USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY so_secrets_org    ON so_secrets    USING (org_id::text = current_setting('app.current_org_id', true));
CREATE POLICY pg_campaigns_org  ON pg_campaigns  USING (org_id::text = current_setting('app.current_org_id', true));
