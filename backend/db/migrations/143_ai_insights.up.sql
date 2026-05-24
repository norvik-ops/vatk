CREATE TABLE ck_ai_insights (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    type         VARCHAR(50) NOT NULL,
    title        TEXT NOT NULL,
    message      TEXT NOT NULL,
    control_id   UUID REFERENCES ck_controls(id) ON DELETE CASCADE,
    risk_id      UUID REFERENCES ck_risks(id)    ON DELETE CASCADE,
    finding_id   UUID REFERENCES vb_findings(id) ON DELETE CASCADE,
    metadata     JSONB,
    urgency      SMALLINT NOT NULL DEFAULT 2,
    dismissed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ck_ai_insights_org_active
    ON ck_ai_insights(org_id, created_at DESC)
    WHERE dismissed_at IS NULL;

ALTER TABLE ck_risks ADD COLUMN ai_narrative TEXT;
ALTER TABLE organizations ADD COLUMN ai_weekly_digest_enabled BOOLEAN NOT NULL DEFAULT false;
