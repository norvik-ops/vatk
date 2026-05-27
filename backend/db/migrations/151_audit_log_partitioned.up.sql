-- 151_audit_log_partitioned.up.sql
--
-- Convert audit_log from a monolithic table into a range-partitioned table
-- keyed on created_at. Each year gets its own partition; rows that fall
-- outside the explicitly-listed ranges go to the DEFAULT partition.
--
-- Why partition?
--   * Insert-side: B-tree indexes on the org_id+created_at composite stay
--     small per-partition; insertion latency does not balloon as the
--     table grows past tens-of-millions of rows.
--   * Read-side: org-scoped time-window queries (the dominant access
--     pattern from /api/v1/audit/...) hit only the touched partitions.
--   * Maintenance-side: monthly export/archival/drop of old partitions
--     becomes a one-line ALTER TABLE ... DETACH operation instead of a
--     long DELETE that locks the live table.
--
-- Constraints we accept:
--   * Postgres requires the partition key to be part of every UNIQUE
--     constraint on the table — so the PRIMARY KEY is now (id, created_at)
--     instead of just (id). Inserts and updates already write both columns,
--     so application code does not need to change.  Foreign keys into
--     audit_log do not exist (verified by grep on the migrations folder),
--     so this PK change is fully backwards-compatible.
--   * The audit_logs VIEW from migration 085 has to be re-created after
--     the table swap because DROP TABLE drops dependent views via CASCADE.
--
-- See ADR-0045 for the design rationale and an alternative analysis.
--
-- Audit P1-2 closure (audit_log unbounded UUID-PK growth).

-- Recreate the audit_logs view after the table swap.
DROP VIEW IF EXISTS audit_logs;

-- The partitioned replacement table. Shape matches the current schema
-- (064 + 011 + 064-chain-cols from migration 149); only the PRIMARY KEY
-- gains created_at to satisfy the partition-key requirement.
CREATE TABLE audit_log_new (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
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
    entry_hash    BYTEA,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- One partition per year, plus a DEFAULT for anything outside the listed
-- ranges. Yearly granularity is plenty for the audit-volume range we
-- expect from per-org self-hosted installs; if a customer ever ramps to
-- monthly granularity, a follow-up migration can split the year-partitions
-- via DETACH/ATTACH (zero downtime).
CREATE TABLE audit_log_2025    PARTITION OF audit_log_new FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
CREATE TABLE audit_log_2026    PARTITION OF audit_log_new FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE audit_log_2027    PARTITION OF audit_log_new FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');
CREATE TABLE audit_log_2028    PARTITION OF audit_log_new FOR VALUES FROM ('2028-01-01') TO ('2029-01-01');
CREATE TABLE audit_log_default PARTITION OF audit_log_new DEFAULT;

-- Copy every row from the legacy table. The PG planner can run this as a
-- straight forward INSERT INTO ... SELECT * FROM ...; partition routing
-- happens row-by-row based on created_at.
INSERT INTO audit_log_new
    (id, org_id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, created_at, prev_hash, entry_hash)
SELECT
    id, org_id, user_id, user_email, action, resource_type, resource_id, resource_name, details, ip_address, created_at, prev_hash, entry_hash
FROM audit_log;

-- Swap the old table out, the partitioned one in.
DROP TABLE audit_log CASCADE;
ALTER TABLE audit_log_new RENAME TO audit_log;

-- Re-create the supporting indexes. CONCURRENTLY is intentionally omitted
-- (golang-migrate wraps every migration in a transaction; see CLAUDE.md).
CREATE INDEX audit_log_org_idx       ON audit_log(org_id, created_at DESC);
CREATE INDEX audit_log_user_idx      ON audit_log(user_id, created_at DESC);
CREATE INDEX audit_log_resource_idx  ON audit_log(resource_type, resource_id);
CREATE INDEX audit_log_org_chain_idx ON audit_log(org_id, created_at, id);
CREATE INDEX idx_audit_log_org_time  ON audit_log(org_id, created_at DESC);

-- Re-create the audit_logs backwards-compat view (migration 085).
CREATE VIEW audit_logs AS
SELECT
    id,
    org_id,
    user_id,
    action,
    resource_type,
    resource_id,
    ip_address,
    NULL::text    AS user_agent,
    NULL::int     AS status_code,
    created_at    AS timestamp
FROM audit_log;
