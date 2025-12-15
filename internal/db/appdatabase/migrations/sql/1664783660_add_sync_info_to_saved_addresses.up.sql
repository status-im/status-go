ALTER TABLE saved_addresses ADD COLUMN removed BOOLEAN NOT NULL DEFAULT 0;
-- Represents wall clock time as unixepoch timestamp
ALTER TABLE saved_addresses ADD COLUMN update_clock INT NOT NULL DEFAULT 0;
-- Update using the current timestamp to deconflict devices already in sync. Wins the last one to update
UPDATE saved_addresses SET update_clock = (CAST(strftime('%s', 'now') AS INT));