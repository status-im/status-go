CREATE TABLE communities_requests_to_join  (
  id BLOB NOT NULL,
  public_key VARCHAR NOT NULL,
  clock INT NOT NULL,
  ens_name VARCHAR NOT NULL DEFAULT "",
  chat_id VARCHAR NOT NULL DEFAULT "",
  community_id BLOB NOT NULL,
  state INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id) ON CONFLICT REPLACE
);


CREATE TABLE ens_verification_records (
  public_key VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  verified BOOLEAN NOT NULL DEFAULT FALSE,
  verified_at INT NOT NULL DEFAULT 0,
  clock INT NOT NULL DEFAULT 0,
  verification_retries INT NOT NULL DEFAULT 0,
  next_retry INT NOT NULL DEFAULT 0,
  PRIMARY KEY (public_key) ON CONFLICT REPLACE
);

INSERT INTO ens_verification_records (public_key, name, verified, verified_at, clock) SELECT id, name, ens_verified, ens_verified_at, ens_verified_at FROM contacts WHERE ens_verified;
