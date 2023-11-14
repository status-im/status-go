CREATE TABLE IF NOT EXISTS curated_communities (
    community_id TEXT PRIMARY KEY,
    featured BOOLEAN NOT NULL DEFAULT FALSE
);
