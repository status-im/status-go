-- One entry per token, currency
CREATE TABLE IF NOT EXISTS price_cache (
    token VARCHAR NOT NULL,
    currency VARCHAR NOT NULL,
    price REAL NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS price_cache_identify_entry ON price_cache (token, currency);