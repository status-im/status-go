ALTER TABLE settings ADD COLUMN backup_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN last_backup INT NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN backup_fetched BOOLEAN DEFAULT FALSE;
UPDATE settings SET backup_enabled = 1;
UPDATE settings SET backup_fetched = 0;
