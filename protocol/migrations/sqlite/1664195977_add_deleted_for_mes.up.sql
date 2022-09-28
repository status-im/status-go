ALTER TABLE user_messages ADD COLUMN deleted_for_me BOOL DEFAULT FALSE;

CREATE TABLE user_messages_deleted_for_mes (
  clock INTEGER NOT NULL,
  message_id VARCHAR NOT NULL,
  source VARCHAR NOT NULL,
  id VARCHAR NOT NULL,
  PRIMARY KEY(id)
);

CREATE INDEX user_messages_deleted_for_mes_message_id_source ON user_messages_edits(message_id, source);
