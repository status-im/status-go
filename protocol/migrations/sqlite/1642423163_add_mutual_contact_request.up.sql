ALTER TABLE contacts ADD COLUMN sent_contact_request_signature BLOB;
ALTER TABLE contacts ADD COLUMN received_contact_request_signature BLOB;
ALTER TABLE contacts ADD COLUMN contact_message_id VARCHAR;
