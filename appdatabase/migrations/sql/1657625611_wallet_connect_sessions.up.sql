CREATE TABLE IF NOT EXISTS wallet_connect_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    peer_id TEXT,
    connector_info TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
