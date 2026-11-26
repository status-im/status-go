CREATE TABLE community_member_delete_all_clocks (
    community_id TEXT NOT NULL,
    member_id TEXT NOT NULL,
    clock INTEGER NOT NULL,
    PRIMARY KEY (community_id, member_id)
) WITHOUT ROWID;
