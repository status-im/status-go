-- connector_permissions table stores individual permissions for each dApp connection
-- This table implements EIP-2255 Wallet Permissions System
-- Each permission has caveats (restrictions) stored as JSON

CREATE TABLE IF NOT EXISTS connector_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT '',
    parent_capability TEXT NOT NULL,
    caveats TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL,
    FOREIGN KEY (url, client_id) REFERENCES connector_dapps(url, client_id) ON DELETE CASCADE,
    UNIQUE(url, client_id, parent_capability)
);

CREATE INDEX idx_connector_permissions_url_client ON connector_permissions(url, client_id);

