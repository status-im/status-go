CREATE TABLE IF NOT EXISTS communities_check_channel_permission_responses (
  community_id VARCHAR NOT NULL,
  chat_id VARCHAR NOT NULL DEFAULT "",
  view_only_permissions_satisfied BOOLEAN NOT NULL DEFAULT TRUE,
  view_and_post_permissions_satisfied BOOLEAN NOT NULL DEFAULT TRUE,
  view_only_permission_ids TEXT NOT NULL,
  view_and_post_permission_ids TEXT NOT NULL,
  PRIMARY KEY(community_id, chat_id) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS communities_permission_token_criteria_results (
  permission_id VARCHAR NOT NULL DEFAULT "",
  community_id VARCHAR NOT NULL,
  chat_id VARCHAR NOT NULL DEFAULT "",
  criteria VARCHAR NOT NULL DEFAULT "",
  PRIMARY KEY(community_id, chat_id, permission_id) ON CONFLICT REPLACE
);

