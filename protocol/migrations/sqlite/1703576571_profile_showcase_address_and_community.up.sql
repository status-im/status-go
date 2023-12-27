-- Recreate tables for storing current user profile showcase collectibles & assets preferences
DROP TABLE profile_showcase_collectibles_preferences;
CREATE TABLE profile_showcase_collectibles_preferences (
    contract_address TEXT PRIMARY KEY ON CONFLICT REPLACE,
    chain_id TEXT NOT NULL,
    token_id TEXT NOT NULL,
    community_id TEXT DEFAULT "",
    visibility INT NOT NULL DEFAULT 0,
    sort_order INT DEFAULT 0
);

DROP TABLE profile_showcase_assets_preferences;
CREATE TABLE profile_showcase_assets_preferences (
    contract_address TEXT PRIMARY KEY ON CONFLICT REPLACE,
    community_id TEXT DEFAULT "",
    symbol TEXT DEFAULT "",
    visibility INT NOT NULL DEFAULT 0,
    sort_order INT DEFAULT 0
);

-- Recreate tables for storing profile showcase collectibles & assets for each contact
DROP INDEX profile_showcase_collectibles_contact_id;
DROP TABLE profile_showcase_collectibles_contacts;
CREATE TABLE profile_showcase_collectibles_contacts (
    contract_address TEXT NOT NULL,
    chain_id TEXT NOT NULL,
    token_id TEXT NOT NULL,
    community_id TEXT DEFAULT "",
    sort_order INT DEFAULT 0,
    contact_id TEXT NOT NULL,
    PRIMARY KEY (contact_id, chain_id, contract_address, token_id)
);
CREATE INDEX profile_showcase_collectibles_contact_id ON profile_showcase_collectibles_contacts (contact_id);

DROP INDEX profile_showcase_assets_contact_id;
DROP TABLE profile_showcase_assets_contacts;
CREATE TABLE profile_showcase_assets_contacts (
    contract_address TEXT NOT NULL,
    community_id TEXT DEFAULT "",
    symbol TEXT DEFAULT "",
    sort_order INT DEFAULT 0,
    contact_id TEXT NOT NULL,
    PRIMARY KEY (contact_id, contract_address)
);
CREATE INDEX profile_showcase_assets_contact_id ON profile_showcase_assets_contacts (contact_id);
