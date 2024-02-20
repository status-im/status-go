CREATE TABLE IF NOT EXISTS user_messages_remove (
  user_id TEXT NOT NULL,
  community_id TEXT DEFAULT "",
  clock INT NOT NULL DEFAULT 0,
  changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, community_id)
);
