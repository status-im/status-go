ALTER TABLE settings ADD COLUMN backup_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN last_backup INT NOT NULL DEFAULT 0;
UPDATE settings SET backup_enabled = 1;
