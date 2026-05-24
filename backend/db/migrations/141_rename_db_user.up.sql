-- Rename PostgreSQL user sechealth → vakt for clean branding
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sechealth') THEN
    ALTER USER sechealth RENAME TO vakt;
  END IF;
END $$;
