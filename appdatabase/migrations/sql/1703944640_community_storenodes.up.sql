-- To allow many to many relationship between communities and mailservers
CREATE TABLE IF NOT EXISTS community_storenodes (
    community_id BLOB NOT NULL,
    storenode_id VARCHAR NOT NULL,
    clock INT DEFAULT 0,
    PRIMARY KEY (community_id, storenode_id),
    FOREIGN KEY (community_id) REFERENCES communities_communities(id) ON DELETE CASCADE,
    FOREIGN KEY (storenode_id) REFERENCES mailservers(id) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX community_storenodes_storenode_id ON community_storenodes(storenode_id);

-- To avoid querying community-only storenodes
ALTER TABLE mailservers ADD COLUMN community_only BOOLEAN DEFAULT FALSE;
