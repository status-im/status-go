CREATE TABLE IF NOT EXISTS community_grants (
  community_id TEXT PRIMARY KEY NOT NULL,
  grant TEXT DEFAULT "",
  clock INT NOT NULL DEFAULT 0
);
