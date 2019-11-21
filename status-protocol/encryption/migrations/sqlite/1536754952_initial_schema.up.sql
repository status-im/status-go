CREATE TABLE sessions (
  dhr BLOB,
  dhs_public BLOB,
  dhs_private BLOB,
  root_chain_key BLOB,
  send_chain_key BLOB,
  send_chain_n INTEGER,
  recv_chain_key BLOB,
  recv_chain_n INTEGER,
  step INTEGER,
  pn  INTEGER,
  id BLOB NOT NULL PRIMARY KEY,
  UNIQUE(id) ON CONFLICT REPLACE
);

CREATE TABLE keys (
  public_key BLOB NOT NULL,
  msg_num INTEGER,
  message_key BLOB NOT NULL,
  UNIQUE (msg_num, message_key) ON CONFLICT REPLACE
);

CREATE TABLE bundles (
  identity BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  private_key BLOB,
  signed_pre_key BLOB NOT NULL PRIMARY KEY ON CONFLICT IGNORE,
  timestamp UNSIGNED BIG INT NOT NULL,
  expired BOOLEAN DEFAULT 0
);

CREATE TABLE ratchet_info (
  bundle_id BLOB NOT NULL,
  ephemeral_key BLOB,
  identity BLOB NOT NULL,
  symmetric_key BLOB NOT NULL,
  installation_id TEXT NOT NULL,
  UNIQUE(bundle_id, identity) ON CONFLICT REPLACE,
  FOREIGN KEY (bundle_id) REFERENCES bundles(signed_pre_key)
);
