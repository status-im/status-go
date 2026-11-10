CREATE TABLE sds_received_messages (
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    received_at INTEGER NOT NULL,
    PRIMARY KEY (channel_id, message_id)
) WITHOUT ROWID;

CREATE INDEX sds_received_messages_received_at ON sds_received_messages (received_at);
