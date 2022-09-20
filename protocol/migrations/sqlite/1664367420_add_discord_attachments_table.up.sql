CREATE TABLE IF NOT EXISTS discord_message_attachments (
  id TEXT PRIMARY KEY NOT NULL,
  discord_message_id TEXT NULL,
  url TEXT NOT NULL,
  file_name TEXT NOT NULL,
  file_size_bytes INT NOT NULL,
  payload BLOB,
  content_type TEXT
) WITHOUT ROWID;

CREATE INDEX dm_attachments_messages_idx ON discord_message_attachments (discord_message_id);

