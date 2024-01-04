ALTER TABLE saved_addresses ADD COLUMN color TEXT DEFAULT "primary";
UPDATE saved_addresses SET color = "primary";