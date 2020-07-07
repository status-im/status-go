CREATE TABLE IF NOT EXISTS push_notification_client_servers (
  public_key BLOB NOT NULL,
  registered BOOLEAN DEFAULT FALSE,
  registered_at INT NOT NULL DEFAULT 0,
  UNIQUE(public_key) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS push_notification_client_info (
  public_key BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  access_token TEXT NOT NULL,
  UNIQUE(public_key, installation_id) ON CONFLICT REPLACE
);

CREATE INDEX idx_push_notification_client_info_public_key ON push_notification_client_info(public_key, installation_id);
