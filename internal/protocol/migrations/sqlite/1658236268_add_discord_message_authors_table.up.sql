CREATE TABLE IF NOT EXISTS discord_message_authors (
  id TEXT PRIMARY KEY NOT NULL,
  name TEXT NOT NULL,
  discriminator TEXT NOT NULL,
  nickname TEXT,
  avatar_url TEXT
) WITHOUT ROWID;

