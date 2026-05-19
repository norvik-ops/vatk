-- 103_control_test_scheduler.up.sql
-- Adds test scheduling fields to controls and expands ck_capas source_type to include 'control_test'.

-- Expand ck_capas source_type CHECK constraint to allow 'control_test'.
ALTER TABLE ck_capas
  DROP CONSTRAINT IF EXISTS ck_capas_source_type_check;
ALTER TABLE ck_capas
  ADD CONSTRAINT ck_capas_source_type_check
    CHECK (source_type IN ('audit','incident','risk','manual','control_test'));

ALTER TABLE ck_controls
  ADD COLUMN IF NOT EXISTS test_interval_days  INT,
  ADD COLUMN IF NOT EXISTS last_tested_at      TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS next_test_due_at    TIMESTAMPTZ;

-- next_test_due_at is computed by the application as:
--   last_tested_at + test_interval_days * INTERVAL '1 day'
-- (GENERATED ALWAYS AS is not possible because timestamptz + interval is STABLE, not IMMUTABLE)

CREATE INDEX IF NOT EXISTS idx_ck_controls_next_test
  ON ck_controls(org_id, next_test_due_at)
  WHERE next_test_due_at IS NOT NULL AND manual_status != 'not_applicable';
