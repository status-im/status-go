-- clean up currency_format_cache table
-- use key instead of symbol column
ALTER TABLE currency_format_cache RENAME TO currency_format_cache_old;

CREATE TABLE IF NOT EXISTS currency_format_cache (
    key VARCHAR NOT NULL,
    symbol VARCHAR NOT NULL,
    display_decimals INT NOT NULL,
    strip_trailing_zeroes BOOLEAN NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS currency_format_cache_identify_entry ON currency_format_cache (key);

DROP TABLE currency_format_cache_old;