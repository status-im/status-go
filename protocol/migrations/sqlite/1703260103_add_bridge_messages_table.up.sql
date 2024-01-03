CREATE TABLE IF NOT EXISTS bridge_messages (
  user_messages_id TEXT PRIMARY KEY NOT NULL,
  bridge_name TEXT NOT NULL,
  user_name TEXT NOT NULL,
  content TEXT NOT NULL
);

CREATE INDEX idx_bridge_messages_user_messages_id
ON bridge_messages (user_messages_id);
