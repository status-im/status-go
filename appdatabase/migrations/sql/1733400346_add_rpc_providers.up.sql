CREATE TABLE IF NOT EXISTS rpc_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,                -- Unique provider ID (sorting)
    chain_id INTEGER NOT NULL CHECK (chain_id > 0),      -- Chain ID for the network
    name TEXT NOT NULL CHECK (LENGTH(name) > 0),         -- Provider name
    url TEXT NOT NULL CHECK (LENGTH(url) > 0),           -- Provider URL
    enable_rps_limiter BOOLEAN NOT NULL DEFAULT FALSE,   -- Enable RPS limiter
    type TEXT NOT NULL DEFAULT 'user',                   -- Provider type: embedded-proxy, embedded-direct, user
    enabled BOOLEAN NOT NULL DEFAULT TRUE,               -- Whether the provider is active or not
    auth_type TEXT NOT NULL DEFAULT 'no-auth',           -- Authentication type: no-auth, basic-auth, token-auth
    auth_login TEXT,                                     -- BasicAuth login (nullable)
    auth_password TEXT,                                  -- Password for BasicAuth (nullable)
    auth_token TEXT                                      -- Token for TokenAuth (nullable)
);
