CREATE TABLE "token_preferences" (
	"key" TEXT NOT NULL,
	"position" INTEGER NOT NULL DEFAULT -1,
	"group_position" INTEGER NOT NULL DEFAULT -1,
	"visible" BOOLEAN NOT NULL DEFAULT TRUE,
	"community_id" TEXT NOT NULL DEFAULT '',
	"testnet" BOOLEAN NOT NULL DEFAULT FALSE,
	PRIMARY KEY("key","testnet")
);

ALTER TABLE settings ADD COLUMN wallet_token_preferences_change_clock INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN wallet_token_preferences_group_by_community BOOLEAN NOT NULL DEFAULT FALSE;