ALTER TABLE message RENAME TO message_old;
DROP INDEX i_ts;
DROP INDEX i_query;

CREATE TABLE message (
	id BYTEA,
	receiverTimestamp BIGINT NOT NULL,
	senderTimestamp BIGINT NOT NULL,
	contentTopic BYTEA NOT NULL,
	pubsubTopic BYTEA NOT NULL,
	payload BYTEA,
	version INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (id, pubsubTopic)
);

CREATE INDEX message_senderTimestamp ON message(senderTimestamp);
CREATE INDEX message_receiverTimestamp ON message(receiverTimestamp);
CREATE INDEX i_msg_1 ON message(contentTopic ASC, pubsubTopic ASC, senderTimestamp ASC, id ASC);
CREATE INDEX i_msg_2 ON message(contentTopic DESC, pubsubTopic DESC, senderTimestamp DESC, id DESC);

INSERT INTO message(id, receiverTimestamp, senderTimestamp, contentTopic, pubsubTopic, payload, version)
SELECT id, storedAt, timestamp, contentTOpic, pubsubTopic, payload, version
FROM message_old;

DROP TABLE message_old;
