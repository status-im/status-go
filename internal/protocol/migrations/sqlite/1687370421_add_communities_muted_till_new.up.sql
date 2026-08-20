CREATE TABLE communities_copy (
    id BLOB NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
    private_key BLOB,
    description BLOB NOT NULL,
    joined BOOLEAN NOT NULL DEFAULT FALSE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    spectated BOOLEAN NOT NULL DEFAULT FALSE,
    muted BOOLEAN NOT NULL DEFAULT FALSE,
    muted_till TIMESTAMP,
    synced_at TIMESTAMP NOT NULL DEFAULT 0
);

INSERT INTO communities_copy SELECT id, private_key, description, joined, verified, spectated, muted, 0, synced_at FROM communities_communities;

DROP TABLE communities_communities;

ALTER TABLE communities_copy RENAME TO communities_communities;
