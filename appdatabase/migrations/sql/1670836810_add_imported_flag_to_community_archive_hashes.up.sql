ALTER TABLE community_message_archive_hashes ADD COLUMN imported BOOL DEFAULT FALSE;
UPDATE community_message_archive_hashes SET imported = 0;

