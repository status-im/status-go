CREATE TABLE IF NOT EXISTS wallet_connect_pairings (
    topic TEXT PRIMARY KEY NOT NULL,
    expiry_timestamp INTEGER NOT NULL,
    active BOOLEAN NOT NULL,
    app_name TEXT,
    url TEXT,
    description TEXT,
    icon TEXT,
    verified_is_scam BOOLEAN,
    verified_origin TEXT,
    verified_verify_url TEXT,
    verified_validation TEXT
);

CREATE INDEX IF NOT EXISTS idx_expiry ON wallet_connect_pairings (expiry_timestamp,active);
