ALTER TABLE settings ADD COLUMN test_networks_enabled BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE settings SET test_networks_enabled = 0;