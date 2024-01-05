CREATE TABLE IF NOT EXISTS community_storenodes (
    community_id BLOB NOT NULL,
    storenode_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    address VARCHAR NOT NULL,
    password VARCHAR,
    fleet VARCHAR NOT NULL,
    version INT NOT NULL,
    clock INT NOT NULL DEFAULT 0,
    removed BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at INT NOT NULL DEFAULT 0,
    PRIMARY KEY (community_id, storenode_id), -- One to many relationship between communities and storenodes: one community might have multiple storenodes
    FOREIGN KEY (community_id) REFERENCES communities_communities(id) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX community_storenodes_storenode_id ON community_storenodes(storenode_id);