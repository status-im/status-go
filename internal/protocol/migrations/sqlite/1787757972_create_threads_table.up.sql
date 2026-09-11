CREATE TABLE IF NOT EXISTS threads (
  thread_id TEXT PRIMARY KEY NOT NULL,
  chat_id VARCHAR NOT NULL,
  parent_message_id VARCHAR NOT NULL,
  name TEXT NOT NULL
);

CREATE INDEX threads_chat_id_name_idx ON threads(chat_id, name);
