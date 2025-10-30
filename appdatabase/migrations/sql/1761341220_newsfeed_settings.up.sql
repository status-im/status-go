-- This migration creates the same table as in
-- services/newsfeed/migrations/sqlite/1761341220_newsfeed_settings.up.sql
-- This allows us to migrate the data from the settings table to the new table without losing data.
-- Yet the services/newsfeed migration is able to be executed independently.
-- These migrations can be executed in any order.

CREATE TABLE IF NOT EXISTS newsfeed_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN DEFAULT TRUE,
    rss_enabled BOOLEAN DEFAULT TRUE,
    notifications_enabled BOOLEAN DEFAULT FALSE,
    last_fetched_timestamp TIMESTAMP
);

-- Copy data from settings table to newsfeed_settings
INSERT OR REPLACE INTO newsfeed_settings (id,
                                          enabled,
                                          rss_enabled,
                                          notifications_enabled,
                                          last_fetched_timestamp)
SELECT 1,
       news_feed_enabled,
       news_rss_enabled,
       news_notifications_enabled,
       -- We cannot guarantee that last_fetched_timestamp will be initialized correctly
       -- if this migration is run on an older version of the database. So we default to CURRENT_TIMESTAMP.
       CURRENT_TIMESTAMP
FROM settings;

-- Drop columns from settings table
ALTER TABLE settings DROP COLUMN news_feed_enabled;
ALTER TABLE settings DROP COLUMN news_rss_enabled;
ALTER TABLE settings DROP COLUMN news_notifications_enabled;
ALTER TABLE settings DROP COLUMN news_feed_last_fetched_timestamp;
