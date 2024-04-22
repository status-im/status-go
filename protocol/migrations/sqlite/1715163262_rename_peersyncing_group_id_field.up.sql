DROP INDEX IF EXISTS peersyncing_messages_timestamp;

ALTER TABLE peersyncing_messages RENAME COLUMN group_id TO chat_id;

CREATE INDEX peersyncing_messages_timestamp ON peersyncing_messages(chat_id, timestamp);
