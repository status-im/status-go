ALTER TABLE wakuv2_config ADD COLUMN enable_store BOOLEAN DEFAULT false;
ALTER TABLE wakuv2_config ADD COLUMN store_capacity INT;
ALTER TABLE wakuv2_config ADD COLUMN store_seconds INT;

UPDATE wakuv2_config SET enable_store = 0, store_capacity = 0, store_seconds = 0;
