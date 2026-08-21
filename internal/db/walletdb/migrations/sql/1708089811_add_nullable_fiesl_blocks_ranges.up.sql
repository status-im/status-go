DROP TABLE blocks_ranges_sequential;

CREATE TABLE blocks_ranges_sequential (
  network_id UNSIGNED BIGINT NOT NULL,
  address VARCHAR NOT NULL,
  blk_start BIGINT DEFAULT null,
  blk_first BIGINT DEFAULT null,
  blk_last BIGINT DEFAULT null,
  token_blk_start BIGINT DEFAULT null,
  token_blk_first BIGINT DEFAULT null,
  token_blk_last BIGINT DEFAULT null,
  balance_check_hash TEXT DEFAULT "",
  PRIMARY KEY (network_id, address)
) WITHOUT ROWID;
