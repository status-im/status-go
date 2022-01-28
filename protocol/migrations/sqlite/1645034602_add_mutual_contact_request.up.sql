CREATE TABLE contact_requests (
  signing_key VARCHAR NOT NULL,
  contact_key VARCHAR NOT NULL,
  signature BLOB NOT NULL,
  timestamp INT NOT NULL,
  PRIMARY KEY (signing_key, contact_key)
);

ALTER TABLE contacts ADD COLUMN contact_message_id VARCHAR;
