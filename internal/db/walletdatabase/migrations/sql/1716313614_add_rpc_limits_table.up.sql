CREATE TABLE rpc_limits (
    tag TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    period INTEGER NOT NULL,
    max_requests INTEGER NOT NULL,
    counter INTEGER NOT NULL
) WITHOUT ROWID;