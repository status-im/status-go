CREATE TABLE IF NOT EXISTS communities_requests_to_join_revealed_addresses (
  request_id BLOB NOT NULL,
  address TEXT NOT NULL,
  PRIMARY KEY(request_id, address)
);
