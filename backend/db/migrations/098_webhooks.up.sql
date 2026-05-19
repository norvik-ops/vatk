-- 098_webhooks.up.sql
-- Outgoing webhook subscriptions per organisation.

CREATE TABLE webhooks (
  id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id           UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name             TEXT        NOT NULL,
  url              TEXT        NOT NULL,
  secret           TEXT,
  events           TEXT[]      NOT NULL DEFAULT '{}',
  active           BOOLEAN     NOT NULL DEFAULT true,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_triggered_at TIMESTAMPTZ,
  last_status_code  INT
);

CREATE INDEX ON webhooks(org_id) WHERE active = true;
