CREATE TABLE IF NOT EXISTS discord_message_attachments (
  id TEXT PRIMARY KEY NOT NULL,
  discord_message_id TEXT NULL,
  url TEXT NOT NULL,
  file_name TEXT NOT NULL,
  file_size_bytes INT NOT NULL,
  payload BLOB,
  type VARCHAR,
  base64 TEXT DEFAULT ""
) WITHOUT ROWID;
