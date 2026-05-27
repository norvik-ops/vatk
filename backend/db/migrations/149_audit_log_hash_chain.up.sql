-- 149_audit_log_hash_chain.up.sql
--
-- Per-org tamper-evidence chain on the audit log.
--
-- Each new entry stores:
--   prev_hash   — entry_hash of the most-recent row for the same org_id
--   entry_hash  — SHA-256( prev_hash || canonical(this_row) )
--
-- A verifier (cmd/audit-verify) replays the chain per org and flags the
-- first row whose recomputed entry_hash disagrees with the stored value.
-- Rows inserted before this migration carry NULL in both columns; the
-- verifier treats that prefix as "pre-tamper-evident" and starts the
-- chain at the first row that has entry_hash IS NOT NULL.
--
-- Audit-Trail prerequisite for ISO 27001 A.12.4.3 / NIS2 / DORA Art. 11.
-- See ADR-0040.

ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS prev_hash  BYTEA,
    ADD COLUMN IF NOT EXISTS entry_hash BYTEA;

-- Covers the per-org chain replay path used by both the writer (SELECT ...
-- ORDER BY created_at DESC LIMIT 1 FOR UPDATE) and the verifier (scan in
-- created_at ASC). Composite on (org_id, created_at, id) gives a stable
-- ordering even when two rows share a timestamp at micro-second precision.
CREATE INDEX IF NOT EXISTS audit_log_org_chain_idx
    ON audit_log(org_id, created_at, id);
