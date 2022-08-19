CREATE TABLE IF NOT EXISTS store_messages (
	id BLOB,
	receiverTimestamp INTEGER NOT NULL,
	senderTimestamp INTEGER NOT NULL,
	contentTopic BLOB NOT NULL,
	pubsubTopic BLOB NOT NULL,
	payload BLOB,
	version INTEGER NOT NULL DEFAULT 0,
	CONSTRAINT messageIndex PRIMARY KEY (id, pubsubTopic)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS store_message_senderTimestamp ON store_messages(senderTimestamp);
CREATE INDEX IF NOT EXISTS store_message_receiverTimestamp ON store_messages(receiverTimestamp);