CREATE TABLE collapsed_community_categories (
  community_id VARCHAR NOT NULL,
  category_id VARCHAR NOT NULL,
  UNIQUE(community_id, category_id) ON CONFLICT REPLACE
);
