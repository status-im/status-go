-- Each connection can have its own clientId (status-desktop browser, chrome extension, wallet-connect)
-- allowing independent session state management
CREATE TABLE connector_dapps_new (
    url TEXT NOT NULL,
    name TEXT NOT NULL,
    shared_account TEXT NOT NULL,
    chain_id UNSIGNED BIGINT NOT NULL,
    icon_url TEXT,
    client_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (url, client_id)
) WITHOUT ROWID;

-- Migrate existing data with empty client_id for backward compatibility
INSERT INTO connector_dapps_new (url, name, shared_account, chain_id, icon_url, client_id)
SELECT url, name, shared_account, chain_id, icon_url, '' FROM connector_dapps;

DROP TABLE connector_dapps;

ALTER TABLE connector_dapps_new RENAME TO connector_dapps;
