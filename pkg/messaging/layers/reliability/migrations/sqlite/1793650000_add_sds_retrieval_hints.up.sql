CREATE TABLE IF NOT EXISTS sds_retrieval_hints (
  channel_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  envelope_hash BLOB NOT NULL,
  PRIMARY KEY (channel_id, message_id, envelope_hash)
);

CREATE INDEX IF NOT EXISTS sds_retrieval_hints_message_id
  ON sds_retrieval_hints (message_id);
