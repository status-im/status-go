-- To allow many to many relationship between communities and mailservers
CREATE TABLE IF NOT EXISTS community_mailservers (
    community_id BLOB NOT NULL,
    mailserver_id VARCHAR NOT NULL,
    PRIMARY KEY (community_id, mailserver_id),
    FOREIGN KEY (community_id) REFERENCES communities_communities(id) ON DELETE CASCADE,
    FOREIGN KEY (mailserver_id) REFERENCES mailservers(id) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX community_mailservers_mailserver_id ON community_mailservers(mailserver_id);

-- To avoid querying community-only mailservers
ALTER TABLE mailservers ADD COLUMN community_only BOOLEAN DEFAULT FALSE;