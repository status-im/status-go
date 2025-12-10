ALTER TABLE settings_sync_clock ADD COLUMN mnemonic_removed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN mnemonic_removed BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE settings SET mnemonic_removed = (SELECT COUNT(*) > 0 FROM settings WHERE mnemonic IS NULL OR mnemonic = '') WHERE synthetic_id = 'id';
