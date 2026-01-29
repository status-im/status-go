-- Rename magnetlink columns to archive_link for clarity
ALTER TABLE communities_archive_info RENAME COLUMN magnetlink_clock TO archive_link_clock;
ALTER TABLE communities_archive_info RENAME COLUMN last_magnetlink_uri TO last_archive_link;
