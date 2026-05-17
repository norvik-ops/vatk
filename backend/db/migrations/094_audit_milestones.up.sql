-- Migration 092: Certification Timeline & Audit Calendar
-- Stores org-level milestones for audits, certifications, and review deadlines.

CREATE TABLE ck_audit_milestones (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id         UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  framework_id   UUID,       -- nullable: org-wide or framework-specific
  title          TEXT        NOT NULL,
  description    TEXT,
  milestone_date DATE        NOT NULL,
  milestone_type TEXT        NOT NULL CHECK (milestone_type IN (
    'internal_audit', 'external_audit', 'certification_target',
    'review_deadline', 'training_deadline', 'custom'
  )),
  status         TEXT        NOT NULL DEFAULT 'upcoming' CHECK (status IN (
    'upcoming', 'completed', 'missed', 'cancelled'
  )),
  created_by     UUID        REFERENCES users(id),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON ck_audit_milestones(org_id, milestone_date);
