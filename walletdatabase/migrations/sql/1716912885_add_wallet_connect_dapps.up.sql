-- wallet_connect_dapps table keeps track of connected dApps to provide a link to their individual sessions
CREATE TABLE IF NOT EXISTS wallet_connect_dapps (
    url TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    icon_url TEXT
) WITHOUT ROWID;

DROP TABLE wallet_connect_sessions;

-- wallet_connect_sessions table keeps track of connected sessions for each dApp
CREATE TABLE wallet_connect_sessions (
    topic TEXT PRIMARY KEY NOT NULL,
    disconnected BOOLEAN NOT NULL,
    session_json JSON NOT NULL,
    expiry INTEGER NOT NULL,
    created_timestamp INTEGER NOT NULL,
    pairing_topic TEXT NOT NULL,
    test_chains BOOLEAN NOT NULL,
    dapp_url TEXT NOT NULL,
    FOREIGN KEY (dapp_url) REFERENCES wallet_connect_dapps(url)
) WITHOUT ROWID;
