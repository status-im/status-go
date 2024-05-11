ALTER TABLE saved_addresses ADD COLUMN emoji TEXT NOT NULL DEFAULT "";
UPDATE saved_addresses SET emoji = "";
