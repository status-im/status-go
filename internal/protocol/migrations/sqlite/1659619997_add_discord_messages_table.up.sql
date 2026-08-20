ALTER TABLE user_messages ADD COLUMN discord_message_id TEXT DEFAULT "";

CREATE TABLE IF NOT EXISTS discord_messages (
  id TEXT PRIMARY KEY NOT NULL,
  author_id TEXT NOT NULL,
  type VARCHAR NOT NULL,
  timestamp INT NOT NULL,
  timestamp_edited INT,
  content TEXT,
  reference_message_id TEXT,
  reference_channel_id TEXT,
  reference_guild_id TEXT
) WITHOUT ROWID;
