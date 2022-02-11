ALTER TABLE settings ADD COLUMN paperclip_preference TEXT NOT NULL DEFAULT "";
UPDATE settings SET paperclip_preference = "";