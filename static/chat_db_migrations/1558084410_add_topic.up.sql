CREATE TABLE topics (
  identity BLOB NOT NULL PRIMARY KEY ON CONFLICT IGNORE,
  secret BLOB NOT NULL
);

CREATE TABLE topic_installation_ids (
  id TEXT NOT NULL,
  identity_id BLOB NOT NULL,
  UNIQUE(id, identity_id) ON CONFLICT IGNORE,
  FOREIGN KEY (identity_id) REFERENCES topics(identity)
);
