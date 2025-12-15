CREATE TABLE IF NOT EXISTS blocks_ranges_sequential (
    network_id UNSIGNED BIGINT NOT NULL,
    address VARCHAR NOT NULL,
    blk_start BIGINT,
    blk_first BIGINT NOT NULL,
    blk_last BIGINT NOT NULL,
    PRIMARY KEY (network_id, address)
) WITHOUT ROWID;
