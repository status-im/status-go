-- delete all from currency_format_cache table. only once, since we've changed the format type.
DELETE FROM currency_format_cache;

ALTER TABLE currency_format_cache RENAME COLUMN symbol TO token_key;