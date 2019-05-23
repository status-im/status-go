CREATE TABLE secrets (
  identity BLOB NOT NULL PRIMARY KEY ON CONFLICT IGNORE,
  secret BLOB NOT NULL
);

CREATE TABLE secret_installation_ids (
  id TEXT NOT NULL,
  identity_id BLOB NOT NULL,
  UNIQUE(id, identity_id) ON CONFLICT IGNORE,
  FOREIGN KEY (identity_id) REFERENCES secrets(identity)
);
