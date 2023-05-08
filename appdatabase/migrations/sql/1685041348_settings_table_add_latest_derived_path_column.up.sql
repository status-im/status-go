ALTER TABLE settings ADD COLUMN latest_derived_path INT NOT NULL DEFAULT "0";
UPDATE settings SET gif_api_key = 0;
