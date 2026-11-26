ALTER TABLE processed_sds_messages ADD COLUMN processed_at INTEGER NOT NULL DEFAULT 0;

-- Existing markers have no timestamp. Give them a full retention window after the upgrade.
UPDATE processed_sds_messages
SET processed_at = CAST(strftime('%s', 'now') AS INTEGER) * 1000;

CREATE INDEX idx_processed_sds_messages_processed_at ON processed_sds_messages (processed_at);
