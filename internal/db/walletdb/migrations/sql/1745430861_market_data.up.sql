CREATE TABLE IF NOT EXISTS market_data (
    id TEXT PRIMARY KEY,               -- Unique provider ID
    symbol TEXT NOT NULL CHECK (LENGTH(symbol) > 0),    -- cryptocurrency symbol
    current_price REAL NOT NULL,                        -- Current price of the cryptocurrency
    market_cap REAL NOT NULL,                           -- Market capitalization
    total_volume REAL NOT NULL,                         -- Total volume traded
    price_change_percentage_24h REAL NOT NULL           -- Price change percentage
);
