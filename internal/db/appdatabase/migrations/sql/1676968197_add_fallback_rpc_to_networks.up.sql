ALTER TABLE networks ADD COLUMN fallback_url VARCHAR NOT NULL DEFAULT "";
UPDATE networks SET fallback_url = "";