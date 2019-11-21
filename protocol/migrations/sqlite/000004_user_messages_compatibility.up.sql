-- It's important that this table has rowid as we rely on it
-- when implementing infinite-scroll.
CREATE TABLE IF NOT EXISTS user_messages_legacy (
    id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
    whisper_timestamp INTEGER NOT NULL,
    source TEXT NOT NULL,
    destination BLOB,
    content VARCHAR NOT NULL,
    content_type VARCHAR NOT NULL,
    username VARCHAR,
    timestamp INT NOT NULL,
    chat_id VARCHAR NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    reply_to VARCHAR,
    message_type VARCHAR,
    message_status VARCHAR,
    clock_value INT NOT NULL,
    show BOOLEAN NOT NULL DEFAULT TRUE,
    seen BOOLEAN NOT NULL DEFAULT FALSE,
    outgoing_status VARCHAR
);

CREATE INDEX idx_source ON user_messages_legacy(source);
CREATE INDEX idx_search_by_chat_id ON  user_messages_legacy(
    substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id, chat_id
);
