-- One entry per symbol
CREATE TABLE IF NOT EXISTS currency_format_cache (
    symbol VARCHAR NOT NULL,
    display_decimals INT NOT NULL,
    strip_trailing_zeroes BOOLEAN NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS currency_format_cache_identify_entry ON currency_format_cache (symbol);