ALTER TABLE settings ADD COLUMN news_feed_last_fetched_timestamp TIMESTAMP;
UPDATE settings SET news_feed_last_fetched_timestamp = CURRENT_TIMESTAMP WHERE news_feed_last_fetched_timestamp IS NULL;
