CREATE TABLE IF NOT EXISTS communities_requests_to_leave (
  id BLOB NOT NULL,
  public_key VARCHAR NOT NULL,
  clock INT NOT NULL,
  community_id BLOB NOT NULL,
  PRIMARY KEY (id) ON CONFLICT REPLACE
);
