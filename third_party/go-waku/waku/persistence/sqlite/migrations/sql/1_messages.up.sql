CREATE TABLE IF NOT EXISTS message (
	id BLOB,
	receiverTimestamp INTEGER NOT NULL,
	senderTimestamp INTEGER NOT NULL,
	contentTopic BLOB NOT NULL,
	pubsubTopic BLOB NOT NULL,
	payload BLOB,
	version INTEGER NOT NULL DEFAULT 0,
	CONSTRAINT messageIndex PRIMARY KEY (id, pubsubTopic)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS message_senderTimestamp ON message(senderTimestamp);
CREATE INDEX IF NOT EXISTS message_receiverTimestamp ON message(receiverTimestamp);