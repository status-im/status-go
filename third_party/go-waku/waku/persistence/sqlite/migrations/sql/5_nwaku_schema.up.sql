ALTER TABLE message RENAME TO message_old;

DROP INDEX message_senderTimestamp;
DROP INDEX message_receiverTimestamp;
DROP INDEX i_msg_1;
DROP INDEX i_msg_2;

CREATE TABLE message (
  pubsubTopic BLOB NOT NULL,
  contentTopic BLOB NOT NULL,
  payload BLOB,
  version INTEGER NOT NULL,
  timestamp INTEGER NOT NULL,
  id BLOB,
  messageHash BLOB, -- Newly added, this will be populated with a counter value
  storedAt INTEGER NOT NULL,
  CONSTRAINT messageIndex PRIMARY KEY (messageHash)
) WITHOUT ROWID;
CREATE INDEX i_ts ON Message (storedAt);
CREATE INDEX i_query ON Message (contentTopic, pubsubTopic, storedAt, id);

INSERT INTO message(pubsubTopic, contentTopic, payload, version, timestamp, id, messageHash, storedAt)
SELECT pubsubTopic, contentTopic, payload, version, senderTimestamp, id, id, receiverTimestamp
FROM message_old;

DROP TABLE message_old;
