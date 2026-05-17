-- 4-Augen-Prinzip: Control status change approval workflow
-- Migration 092

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS approval_required BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE ck_control_approvals (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  control_id       UUID NOT NULL,
  requested_by     UUID NOT NULL REFERENCES users(id),
  requested_status TEXT NOT NULL,
  current_status   TEXT NOT NULL,
  comment          TEXT,
  status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
  reviewed_by      UUID REFERENCES users(id),
  reviewed_at      TIMESTAMPTZ,
  review_comment   TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON ck_control_approvals(org_id, control_id, status);
CREATE INDEX ON ck_control_approvals(org_id, status) WHERE status = 'pending';
