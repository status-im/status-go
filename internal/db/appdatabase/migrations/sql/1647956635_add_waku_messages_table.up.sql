CREATE TABLE waku_messages (
  sig BLOB NOT NULL,
  timestamp INT NOT NULL,
  topic TEXT NOT NULL,
  payload BLOB NOT NULL,
  padding BLOB NOT NULL,
  hash TEXT PRIMARY KEY NOT NULL
);

CREATE INDEX waku_messages_timestamp_topic ON waku_messages (timestamp, topic);
