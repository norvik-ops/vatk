-- Shared cross-module comments table for findings and controls.
-- Findings live in SecPulse, controls in SecVitals; this table allows both
-- to carry threaded discussion without a direct cross-module DB dependency.

CREATE TABLE comments (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  entity_type TEXT        NOT NULL CHECK (entity_type IN ('finding', 'control')),
  entity_id   UUID        NOT NULL,
  author_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content     TEXT        NOT NULL CHECK (char_length(content) BETWEEN 1 AND 4000),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ
);

CREATE INDEX ON comments(org_id, entity_type, entity_id) WHERE deleted_at IS NULL;
