CREATE TABLE peersyncing_messages (
  id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
  type INT NOT NULL,
  group_id VARCHAR NOT NULL,
  payload BLOB NOT NULL,
  timestamp INT NOT NULL
);

CREATE INDEX peersyncing_messages_timestamp ON peersyncing_messages(group_id, timestamp);
