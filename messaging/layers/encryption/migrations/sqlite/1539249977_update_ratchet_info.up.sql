DELETE FROM sessions;
DELETE FROM keys;
DROP TABLE ratchet_info;

CREATE TABLE ratchet_info_v2 (
  bundle_id BLOB NOT NULL,
  ephemeral_key BLOB,
  identity BLOB NOT NULL,
  symmetric_key BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  UNIQUE(bundle_id, identity, installation_id) ON CONFLICT REPLACE,
  FOREIGN KEY (bundle_id) REFERENCES bundles(signed_pre_key)
);
