-- Migration 146: vb_risk_trend_snapshots — daily materialized risk data per org.
-- A background worker writes one row per org per day; the dashboard reads from
-- this table instead of running generate_series × vb_findings at request time.
CREATE TABLE IF NOT EXISTS vb_risk_trend_snapshots (
    org_id          UUID        NOT NULL,
    snapshot_date   DATE        NOT NULL,
    open_count      INT         NOT NULL DEFAULT 0,
    critical_count  INT         NOT NULL DEFAULT 0,
    total_risk_score FLOAT8     NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_risk_trend_snap_org_date
    ON vb_risk_trend_snapshots (org_id, snapshot_date DESC);
