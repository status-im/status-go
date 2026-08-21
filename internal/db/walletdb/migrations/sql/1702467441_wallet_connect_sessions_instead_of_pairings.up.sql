CREATE TABLE IF NOT EXISTS wallet_connect_sessions (
    topic TEXT PRIMARY KEY NOT NULL,
    pairing_topic TEXT NOT NULL,
    expiry INTEGER NOT NULL,
    active BOOLEAN NOT NULL,
    dapp_name TEXT,
    dapp_url TEXT,
    dapp_description TEXT,
    dapp_icon TEXT,
    dapp_verify_url TEXT,
    dapp_publicKey TEXT
);

DROP TABLE wallet_connect_pairings;