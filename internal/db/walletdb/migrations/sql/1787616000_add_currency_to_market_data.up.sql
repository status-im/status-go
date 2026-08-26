-- Leaderboard market data is stored in the currency it was fetched in.
-- Rows written before this migration are USD, because the proxy was always
-- queried without a conversion currency.
ALTER TABLE market_data ADD COLUMN currency TEXT NOT NULL DEFAULT 'usd';
