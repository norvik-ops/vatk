CREATE TABLE ck_control_changelog (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  control_id  UUID NOT NULL REFERENCES ck_controls(id) ON DELETE CASCADE,
  org_id      UUID NOT NULL,
  user_id     UUID,
  user_email  TEXT,
  field       TEXT NOT NULL,
  old_value   TEXT,
  new_value   TEXT,
  changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON ck_control_changelog(control_id, changed_at DESC);
