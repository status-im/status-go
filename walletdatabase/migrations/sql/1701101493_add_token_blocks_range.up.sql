-- Add columns
ALTER TABLE blocks_ranges_sequential ADD COLUMN token_blk_start BIGINT DEFAULT 0;
ALTER TABLE blocks_ranges_sequential ADD COLUMN token_blk_first BIGINT DEFAULT 0;
ALTER TABLE blocks_ranges_sequential ADD COLUMN token_blk_last BIGINT DEFAULT 0;

-- Copy values
UPDATE blocks_ranges_sequential SET token_blk_start = blk_start;
UPDATE blocks_ranges_sequential SET token_blk_first = blk_first;
UPDATE blocks_ranges_sequential SET token_blk_last = blk_last;
