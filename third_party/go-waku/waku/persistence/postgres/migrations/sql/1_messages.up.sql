CREATE TABLE IF NOT EXISTS message (
	id BYTEA,
	receiverTimestamp BIGINT NOT NULL,
	senderTimestamp BIGINT NOT NULL,
	contentTopic BYTEA NOT NULL,
	pubsubTopic BYTEA NOT NULL,
	payload BYTEA,
	version INTEGER NOT NULL DEFAULT 0,
	CONSTRAINT messageIndex PRIMARY KEY (id, pubsubTopic)
);

CREATE INDEX IF NOT EXISTS message_senderTimestamp ON message(senderTimestamp);
CREATE INDEX IF NOT EXISTS message_receiverTimestamp ON message(receiverTimestamp);