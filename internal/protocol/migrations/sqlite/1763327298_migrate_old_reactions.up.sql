-- 1763327299_populate_emoji_from_emoji_id.up.sql
-- Populate missing emoji values from legacy emoji_id
UPDATE emoji_reactions
SET emoji = CASE emoji_id
    -- adjust numeric IDs below if they differ from your protobuf enum values
    WHEN 1 THEN '2764'    -- LOVE
    WHEN 2 THEN '1f44d'   -- THUMBS_UP
    WHEN 3 THEN '1f44e'   -- THUMBS_DOWN
    WHEN 4 THEN '1f602'   -- LAUGH
    WHEN 5 THEN '1f622'   -- SAD
    WHEN 6 THEN '1f620'   -- ANGRY
    ELSE ''
END
WHERE emoji IS NULL OR emoji = '';