CREATE TABLE IF NOT EXISTS community_storenodes (
    community_id BLOB NOT NULL,
    storenode_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    address VARCHAR NOT NULL,
    fleet VARCHAR NOT NULL,
    version INT NOT NULL,
    clock INT NOT NULL DEFAULT 0,
    removed BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at INT NOT NULL DEFAULT 0,
    PRIMARY KEY (community_id, storenode_id), -- One to many relationship between communities and storenodes: one community might have multiple storenodes
) WITHOUT ROWID;