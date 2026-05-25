-- Migration 144: Lock registration after initial setup.
-- After the first organisation is created, POST /auth/register is disabled.
-- The application enforces this in service code (not via a column flag so that
-- no existing row is touched and the Demo-flow (RunEphemeral) is unaffected).
--
-- This migration is intentionally a no-op SQL change — the enforcement lives
-- in auth/service.go. The migration records the intent in the schema history.
SELECT 1; -- no-op placeholder
