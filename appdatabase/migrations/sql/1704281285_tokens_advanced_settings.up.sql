ALTER TABLE settings ADD COLUMN wallet_show_community_asset_when_sending_tokens INTEGER NOT NULL DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN wallet_display_assets_below_balance BOOLEAN NOT NULL DEFAULT FALSE;
-- 9 places reserved for decimals. Default value translates to 0.1
ALTER TABLE settings ADD COLUMN wallet_display_assets_below_balance_threshold UNSIGNED BIGINT NOT NULL DEFAULT 100000000;

ALTER TABLE settings_sync_clock ADD COLUMN wallet_show_community_asset_when_sending_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings_sync_clock ADD COLUMN wallet_display_assets_below_balance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings_sync_clock ADD COLUMN wallet_display_assets_below_balance_threshold INTEGER NOT NULL DEFAULT 0;