-- Migration 145: Composite index on audit_log(org_id, created_at DESC).
-- Eliminates full-table scans on the audit log for per-org timeline queries.
-- CONCURRENTLY omitted intentionally — golang-migrate wraps in a transaction
-- and CONCURRENTLY fails inside a transaction (SQLSTATE 25001).
CREATE INDEX IF NOT EXISTS idx_audit_log_org_time
    ON audit_log (org_id, created_at DESC);
