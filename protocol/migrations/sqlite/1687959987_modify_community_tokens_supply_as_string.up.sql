ALTER TABLE community_tokens ADD COLUMN supply_str TEXT NOT NULL DEFAULT "";

ALTER TABLE community_tokens DROP COLUMN supply;
