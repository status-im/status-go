-- It's important that this table has rowid as we rely on it
-- when implementing infinite-scroll.
CREATE TABLE IF NOT EXISTS user_messages_legacy (
    id VARCHAR PRIMARY KEY ON CONFLICT REPLACE,
    whisper_timestamp INTEGER NOT NULL,
    source TEXT NOT NULL,
    destination BLOB,
    text VARCHAR NOT NULL,
    content_type INT NOT NULL,
    username VARCHAR,
    timestamp INT NOT NULL,
    chat_id VARCHAR NOT NULL,
    local_chat_id VARCHAR NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    response_to VARCHAR,
    message_type INT,
    clock_value INT NOT NULL,
    seen BOOLEAN NOT NULL DEFAULT FALSE,
    outgoing_status VARCHAR,
    parsed_text BLOB,
    raw_payload BLOB,
    sticker_pack INT,
    sticker_hash VARCHAR
);

CREATE INDEX idx_source ON user_messages_legacy(source);
CREATE INDEX idx_search_by_chat_id ON  user_messages_legacy(
    substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id, chat_id
);
