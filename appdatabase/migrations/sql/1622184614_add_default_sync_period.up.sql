ALTER TABLE settings ADD COLUMN default_sync_period INTEGER DEFAULT 86400;
UPDATE settings SET default_sync_period = 86400;

