CREATE TABLE IF NOT EXISTS pin_messages (
  id VARCHAR PRIMARY KEY NOT NULL,
  message_id VARCHAR NOT NULL,
  whisper_timestamp INTEGER NOT NULL,
  chat_id VARCHAR NOT NULL,
  local_chat_id VARCHAR NOT NULL,
  clock_value INT NOT NULL,
  pinned BOOLEAN NOT NULL
);
