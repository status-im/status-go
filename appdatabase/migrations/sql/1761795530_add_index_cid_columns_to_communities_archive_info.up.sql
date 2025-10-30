ALTER TABLE communities_archive_info ADD COLUMN last_index_cid TEXT DEFAULT "";
ALTER TABLE communities_archive_info ADD COLUMN index_cid_clock INTEGER DEFAULT 0;