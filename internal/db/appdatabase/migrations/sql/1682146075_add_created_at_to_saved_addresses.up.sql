ALTER TABLE saved_addresses ADD COLUMN created_at INT DEFAULT 0;
UPDATE saved_addresses SET created_at = 0;