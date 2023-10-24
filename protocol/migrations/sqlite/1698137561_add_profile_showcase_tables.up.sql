CREATE TABLE IF NOT EXISTS profile_showcase_preferences (
    id TEXT PRIMARY KEY ON CONFLICT REPLACE,
    entry_type INT NOT NULL DEFAULT 0,
    visibility INT NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS profile_showcase_contacts (
    contact_id TEXT PRIMARY KEY ON CONFLICT REPLACE,
    entry_id TEXT NOT NULL,
    entry_type INT NOT NULL DEFAULT 0,
    entry_order INT NOT NULL DEFAULT 0
);
