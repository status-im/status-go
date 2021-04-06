CREATE TABLE IF NOT EXISTS pin_messages (
  message_id VARCHAR PRIMARY KEY NOT NULL ON CONFLICT REPLACE,
  whisper_timestamp INTEGER NOT NULL,
  chat_id VARCHAR NOT NULL,
  local_chat_id VARCHAR NOT NULL,
  clock_value INT NOT NULL,
  pinned BOOLEAN NOT NULL
);
