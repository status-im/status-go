CREATE TABLE IF NOT EXISTS communities_permissions (
    community_id TEXT NOT NULL,
    permission_id VARCHAR NOT NULL PRIMARY KEY,
    hide BOOLEAN NOT NULL DEFAULT FALSE,
    is_allowed_to INT NOT NULL DEFAULT 0,
    holds_tokens BOOLEAN NOT NULL DEFAULT FALSE 
);

CREATE TABLE IF NOT EXISTS communities_permissions_chats {
    permission_id VARCHAR NOT NULL,
    chat_id VARCHAR NOT NULL
}