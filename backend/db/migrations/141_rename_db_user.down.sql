DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'vakt') THEN
    ALTER USER vakt RENAME TO sechealth;
  END IF;
END $$;
