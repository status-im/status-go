ALTER TABLE user_messages ADD COLUMN edited_at INTEGER;

CREATE TABLE user_messages_edits (
  clock INTEGER NOT NULL,
  chat_id VARCHAR NOT NULL,
  message_id VARCHAR NOT NULL,
  source VARCHAR NOT NULL,
  text VARCHAR NOT NULL,
  id VARCHAR NOT NULL,
  PRIMARY KEY(id)
);
