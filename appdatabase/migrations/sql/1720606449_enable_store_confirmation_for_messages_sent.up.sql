ALTER TABLE wakuv2_config ADD COLUMN enable_store_confirmation_for_messages_sent BOOLEAN DEFAULT false;

UPDATE wakuv2_config SET enable_store_confirmation_for_messages_sent = true;

