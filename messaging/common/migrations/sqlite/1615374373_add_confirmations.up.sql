CREATE TABLE raw_message_confirmations (
  datasync_id BLOB NOT NULL,
  message_id BLOB NOT NULL,
  public_key BLOB NOT NULL,
  confirmed_at INT NOT NULL DEFAULT 0,
  PRIMARY KEY (message_id, public_key) ON CONFLICT REPLACE
);
