-- This migration creates the same table as in
-- protocol/migrations/sqlite/1761169141_newsfeed_settings.up.sql

CREATE TABLE IF NOT EXISTS newsfeed_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN DEFAULT TRUE,
    rss_enabled BOOLEAN DEFAULT TRUE,
    last_fetched_timestamp TIMESTAMP
);

-- Insert default row only if no row exists yet
-- Row might exist if the original protocol migration was applied.
INSERT OR IGNORE INTO newsfeed_settings (id) VALUES (1);

UPDATE newsfeed_settings SET last_fetched_timestamp = CURRENT_TIMESTAMP WHERE last_fetched_timestamp IS NULL;
