CREATE TABLE IF NOT EXISTS chat_identity_social_links (
  chat_id VARCHAR NOT NULL,
  link_text TEXT,
  link_url TEXT,
  UNIQUE(chat_id, link_text) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS chat_identity_last_received (
  chat_id VARCHAR NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
  clock_value INT NOT NULL
);