CREATE TABLE IF NOT EXISTS push_notification_client_servers (
  public_key BLOB NOT NULL,
  registered BOOLEAN DEFAULT FALSE,
  registered_at INT NOT NULL DEFAULT 0,
  last_retried_at INT NOT NULL DEFAULT 0,
  retry_count INT NOT NULL DEFAULT 0,
  access_token TEXT,
  UNIQUE(public_key) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS push_notification_client_queries (
  public_key BLOB NOT NULL,
  queried_at INT NOT NULL,
  query_id BLOB NOT NULL,
  UNIQUE(public_key,query_id) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS push_notification_client_info (
  public_key BLOB NOT NULL,
  server_public_key BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  access_token TEXT NOT NULL,
  retrieved_at INT NOT NULL,
  UNIQUE(public_key, installation_id, server_public_key) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS push_notification_client_tracked_messages (
  message_id BLOB NOT NULL,
  chat_id TEXT NOT NULL,
  tracked_at INT NOT NULL,
  UNIQUE(message_id) ON CONFLICT IGNORE
  );

CREATE TABLE IF NOT EXISTS push_notification_client_sent_notifications (
  message_id BLOB NOT NULL,
  public_key BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  sent_at INT NOT NULL,
  UNIQUE(message_id, public_key, installation_id)
  );

CREATE TABLE IF NOT EXISTS push_notification_client_registrations (
    registration BLOB NOT NULL,
    contact_ids BLOB,
    synthetic_id INT NOT NULL DEFAULT 0,
    UNIQUE(synthetic_id) ON CONFLICT REPLACE
);

CREATE INDEX idx_push_notification_client_info_public_key ON push_notification_client_info(public_key, installation_id);
