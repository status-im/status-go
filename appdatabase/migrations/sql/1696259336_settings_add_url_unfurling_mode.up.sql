ALTER TABLE settings ADD COLUMN url_unfurling_mode INT NOT NULL DEFAULT 1;
ALTER TABLE settings_sync_clock ADD COLUMN url_unfurling_mode INT NOT NULL DEFAULT 0;