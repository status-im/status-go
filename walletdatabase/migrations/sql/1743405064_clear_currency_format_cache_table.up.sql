-- delete all from currency_format_cache table, only once, since we've changed the format type.
DELETE FROM currency_format_cache;

ALTER TABLE currency_format_cache RENAME COLUMN symbol TO id;

DROP INDEX IF EXISTS currency_format_cache_identify_entry;

CREATE UNIQUE INDEX currency_format_cache_identify_entry ON currency_format_cache (id);

-- delete all from token_lists table, only once cause we want to remove potentially stored token lists that don't have id parameter for token
DELETE FROM token_lists;