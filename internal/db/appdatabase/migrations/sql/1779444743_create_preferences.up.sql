CREATE TABLE IF NOT EXISTS preferences (
    category   TEXT    NOT NULL,
    key        TEXT    NOT NULL,
    value      TEXT    NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (category, key)
) WITHOUT ROWID;
