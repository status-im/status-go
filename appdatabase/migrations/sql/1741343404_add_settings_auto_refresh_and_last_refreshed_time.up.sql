ALTER TABLE settings ADD COLUMN auto_refresh_tokens_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN last_tokens_update TIMESTAMP;

UPDATE settings SET auto_refresh_tokens_enabled = 1;

ALTER TABLE settings_sync_clock ADD COLUMN auto_refresh_tokens_enabled INTEGER NOT NULL DEFAULT 0;