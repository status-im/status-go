CREATE TABLE IF NOT EXISTS bridge_messages (
  user_messages_id TEXT PRIMARY KEY NOT NULL,
  bridge_name TEXT NOT NULL,
  user_name TEXT NOT NULL,
  user_avatar TEXT DEFAULT "",
  user_id TEXT DEFAULT "",
  content TEXT NOT NULL,
  message_id TEXT DEFAULT "",
  parent_message_id TEXT DEFAULT ""
);

CREATE INDEX idx_bridge_messages_user_messages_id
ON bridge_messages (user_messages_id);
