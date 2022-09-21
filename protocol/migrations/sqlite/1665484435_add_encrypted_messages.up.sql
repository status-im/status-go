CREATE TABLE hash_ratchet_encrypted_messages (
  hash BLOB PRIMARY KEY ON CONFLICT REPLACE,
  sig BLOB NOT NULL,
  TTL INT NOT NULL,
  timestamp INT NOT NULL,
  topic BLOB NOT NULL,
  payload BLOB NOT NULL,
  dst BLOB,
  p2p BOOLEAN,
  group_id BLOB NOT NULL,
  key_id INT NOT NULL
);

CREATE INDEX hash_ratchet_encrypted_messages_group_id_key_id ON hash_ratchet_encrypted_messages(group_id, key_id);

