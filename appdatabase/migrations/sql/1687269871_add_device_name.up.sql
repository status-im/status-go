ALTER TABLE settings ADD COLUMN device_name TEXT NOT NULL DEFAULT "";
UPDATE settings SET device_name = "";
