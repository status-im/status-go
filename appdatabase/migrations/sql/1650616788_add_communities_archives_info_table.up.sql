CREATE TABLE IF NOT EXISTS communities_archive_info (
  community_id TEXT PRIMARY KEY ON CONFLICT REPLACE,
  magnetlink_clock INT NOT NULL DEFAULT 0,
  last_message_archive_end_date INT NOT NULL DEFAULT 0
)

