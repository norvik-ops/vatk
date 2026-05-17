DROP TABLE IF EXISTS ck_control_approvals;
ALTER TABLE organizations DROP COLUMN IF EXISTS approval_required;
