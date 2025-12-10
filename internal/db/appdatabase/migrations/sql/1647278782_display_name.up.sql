ALTER TABLE settings ADD COLUMN display_name TEXT NOT NULL DEFAULT "";
UPDATE settings SET display_name = "";
