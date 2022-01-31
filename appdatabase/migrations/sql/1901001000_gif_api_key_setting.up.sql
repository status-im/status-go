ALTER TABLE settings ADD COLUMN gif_api_key TEXT NOT NULL DEFAULT "";
UPDATE settings SET gif_api_key = "";
