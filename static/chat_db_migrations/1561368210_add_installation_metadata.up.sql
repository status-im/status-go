CREATE TABLE installation_metadata  (
  identity BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  device_type TEXT NOT NULL DEFAULT '',
  fcm_token TEXT NOT NULL DEFAULT '',
  UNIQUE(identity, installation_id) ON CONFLICT REPLACE
);
