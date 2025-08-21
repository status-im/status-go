CREATE TABLE installations  (
  identity BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  timestamp UNSIGNED BIG INT NOT NULL,
  enabled BOOLEAN DEFAULT 1,
  UNIQUE(identity, installation_id) ON CONFLICT REPLACE
);
