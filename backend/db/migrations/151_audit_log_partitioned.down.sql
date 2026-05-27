-- Rolls Migration 151 back: collapses the partitioned audit_log back into
-- a single monolithic table. Data is preserved.

DROP VIEW IF EXISTS audit_logs;

CREATE TABLE audit_log_legacy (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email    TEXT,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT,
    resource_name TEXT,
    details       JSONB,
    ip_address    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    prev_hash     BYTEA,
    entry_hash    BYTEA
);

INSERT INTO audit_log_legacy SELECT
    id, org_id, user_id, user_email, action, resource_type, resource_id,
    resource_name, details, ip_address, created_at, prev_hash, entry_hash
FROM audit_log;

DROP TABLE audit_log CASCADE;
ALTER TABLE audit_log_legacy RENAME TO audit_log;

CREATE INDEX audit_log_org_idx       ON audit_log(org_id, created_at DESC);
CREATE INDEX audit_log_user_idx      ON audit_log(user_id, created_at DESC);
CREATE INDEX audit_log_resource_idx  ON audit_log(resource_type, resource_id);
CREATE INDEX audit_log_org_chain_idx ON audit_log(org_id, created_at, id);
CREATE INDEX idx_audit_log_org_time  ON audit_log(org_id, created_at DESC);

CREATE VIEW audit_logs AS
SELECT
    id, org_id, user_id, action, resource_type, resource_id, ip_address,
    NULL::text AS user_agent, NULL::int AS status_code, created_at AS timestamp
FROM audit_log;
