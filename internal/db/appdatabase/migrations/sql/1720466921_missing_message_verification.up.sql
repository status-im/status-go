ALTER TABLE wakuv2_config ADD COLUMN enable_missing_message_verification BOOLEAN DEFAULT false;

UPDATE wakuv2_config SET enable_missing_message_verification = true;

