CREATE TABLE pin_messages_new (
  id VARCHAR PRIMARY KEY NOT NULL,
  message_id VARCHAR NOT NULL,
  whisper_timestamp INTEGER NOT NULL,
  chat_id VARCHAR NOT NULL,
  local_chat_id VARCHAR NOT NULL,
  clock_value INT NOT NULL,
  pinned BOOLEAN NOT NULL,
  pinned_by TEXT,
  FOREIGN KEY (message_id) REFERENCES user_messages(id) ON DELETE CASCADE
);

INSERT INTO pin_messages_new (id, message_id, whisper_timestamp, chat_id, local_chat_id, clock_value, pinned, pinned_by)
SELECT id, message_id, whisper_timestamp, chat_id, local_chat_id, clock_value, pinned, pinned_by
FROM pin_messages;

DROP TABLE pin_messages;

ALTER TABLE pin_messages_new RENAME TO pin_messages;
