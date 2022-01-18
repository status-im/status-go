CREATE TABLE contact_requests (
  signing_key VARCHAR NOT NULL,
  contact_key VARCHAR NOT NULL,
  signature BLOB NOT NULL,
  timestamp INT NOT NULL,
  PRIMARY KEY (signing_key, contact_key)
);

ALTER TABLE contacts ADD COLUMN contact_message_id VARCHAR;
ALTER TABLE contacts ADD COLUMN contact_request_clock INT;
ALTER TABLE user_messages ADD COLUMN contact_request_state INT;

CREATE INDEX contact_request_state ON user_messages(contact_request_state);
