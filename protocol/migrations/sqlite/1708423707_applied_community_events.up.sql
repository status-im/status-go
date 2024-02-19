CREATE TABLE applied_community_events (
    community_id TEXT NOT NULL,
    event_type_id TEXT DEFAULT NULL,
    clock INT NOT NULL,
    PRIMARY KEY (community_id, event_type_id) ON CONFLICT REPLACE
);