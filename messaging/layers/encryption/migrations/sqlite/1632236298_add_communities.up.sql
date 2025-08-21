CREATE TABLE hash_ratchet_encryption (
  group_id BLOB NOT NULL,
  key_id INT NOT NULL,
  key BLOB NOT NULL,
  PRIMARY KEY(group_id, key_id)
);

CREATE UNIQUE INDEX idx_hash_ratchet_enc ON hash_ratchet_encryption(group_id, key_id);

CREATE TABLE hash_ratchet_encryption_cache (
  group_id BLOB NOT NULL,
  key_id int NOT NULL,
  seq_no INTEGER,
  hash BLOB NOT NULL,
  sym_enc_key BLOB,
  FOREIGN KEY(group_id, key_id) REFERENCES hash_ratchet_encryption(group_id, key_id)
);

CREATE UNIQUE INDEX idx_hash_ratchet_enc_cache ON hash_ratchet_encryption_cache(group_id, key_id, seq_no);

