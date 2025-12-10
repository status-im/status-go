CREATE TABLE "collectible_preferences" (
	"type" INTEGER NOT NULL,
	"key" TEXT NOT NULL,
	"position" INTEGER NOT NULL DEFAULT -1,
	"visible" BOOLEAN NOT NULL DEFAULT TRUE,
	"testnet" BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY("type", "key", "testnet")
);

ALTER TABLE settings ADD COLUMN wallet_collectible_preferences_change_clock INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN wallet_collectible_preferences_group_by_collection BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE settings ADD COLUMN wallet_collectible_preferences_group_by_community BOOLEAN NOT NULL DEFAULT FALSE;