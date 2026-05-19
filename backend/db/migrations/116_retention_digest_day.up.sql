ALTER TABLE retention_config ADD COLUMN IF NOT EXISTS digest_day SMALLINT NOT NULL DEFAULT 1;
COMMENT ON COLUMN retention_config.digest_day IS '0=Sonntag … 6=Samstag (cron weekday convention)';
