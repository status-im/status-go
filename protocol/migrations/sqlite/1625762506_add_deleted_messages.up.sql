ALTER TABLE user_messages ADD COLUMN deleted BOOL DEFAULT FALSE;

CREATE TABLE user_messages_deletes (
  clock INTEGER NOT NULL,
  chat_id VARCHAR NOT NULL,
  message_id VARCHAR NOT NULL,
  source VARCHAR NOT NULL,
  id VARCHAR NOT NULL,
  PRIMARY KEY(id)
);

CREATE INDEX user_messages_deletes_message_id_source ON user_messages_edits(message_id, source);
