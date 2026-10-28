CREATE TABLE IF NOT EXISTS sds_channel_metadata (
  channel_id TEXT NOT NULL PRIMARY KEY,
  metadata BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS sds_channel_history (
  channel_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  lamport_timestamp INTEGER NOT NULL,
  message BLOB NOT NULL,
  PRIMARY KEY (channel_id, message_id)
);

CREATE INDEX IF NOT EXISTS sds_channel_history_order
  ON sds_channel_history (channel_id, lamport_timestamp, message_id);
