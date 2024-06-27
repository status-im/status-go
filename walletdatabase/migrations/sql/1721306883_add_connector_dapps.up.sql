-- connector_dapps table keeps track of connected dApps to provide a link to their individual sessions
-- should be aligned with wallet_connect_dapps table

CREATE TABLE IF NOT EXISTS connector_dapps (
    url TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    shared_account TEXT NOT NULL,
    icon_url TEXT
) WITHOUT ROWID;
