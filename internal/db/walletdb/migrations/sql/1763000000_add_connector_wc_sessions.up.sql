-- connector_wc_sessions stores WalletConnect protocol session data
-- WC DApps use client_id = 'walletconnect' in connector_dapps
CREATE TABLE IF NOT EXISTS connector_wc_sessions (
    topic TEXT PRIMARY KEY NOT NULL,
    session_json TEXT NOT NULL,
    expiry INTEGER NOT NULL,
    pairing_topic TEXT NOT NULL,
    dapp_url TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT 'walletconnect',
    created_timestamp INTEGER NOT NULL,
    FOREIGN KEY (dapp_url, client_id) REFERENCES connector_dapps(url, client_id)
) WITHOUT ROWID;
