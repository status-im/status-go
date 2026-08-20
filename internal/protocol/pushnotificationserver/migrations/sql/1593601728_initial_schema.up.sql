CREATE TABLE IF NOT EXISTS push_notification_server_registrations (
  public_key BLOB NOT NULL,
  installation_id VARCHAR NOT NULL,
  version INT NOT NULL,
  registration BLOB,
  UNIQUE(public_key, installation_id) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS push_notification_server_identity (
  private_key BLOB NOT NULL,
  synthetic_id INT NOT NULL DEFAULT 0,
  UNIQUE(synthetic_id)
);

CREATE INDEX idx_push_notification_server_registrations_public_key ON push_notification_server_registrations(public_key);
CREATE INDEX idx_push_notification_server_registrations_public_key_installation_id ON push_notification_server_registrations(public_key, installation_id);

