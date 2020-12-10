CREATE TABLE IF NOT EXISTS chat_identity_last_published (
  chat_id VARCHAR NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
  clock_value INT NOT NULL,
  hash BLOB NOT NULL
);
