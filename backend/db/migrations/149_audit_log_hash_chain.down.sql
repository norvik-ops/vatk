-- Reverse Migration 149.  Drops the hash-chain columns; chain history is
-- lost on rollback.

DROP INDEX IF EXISTS audit_log_org_chain_idx;

ALTER TABLE audit_log
    DROP COLUMN IF EXISTS entry_hash,
    DROP COLUMN IF EXISTS prev_hash;
