-- Revert the rename back to magnetlink
ALTER TABLE communities_archive_info RENAME COLUMN archive_link_clock TO magnetlink_clock;
ALTER TABLE communities_archive_info RENAME COLUMN last_archive_link TO last_magnetlink_uri;
