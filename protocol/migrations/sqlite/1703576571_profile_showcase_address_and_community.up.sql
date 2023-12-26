ALTER TABLE profile_showcase_collectibles_preferences
ADD COLUMN community_id TEXT DEFAULT "";

ALTER TABLE profile_showcase_assets_preferences
ADD COLUMN contract_address TEXT DEFAULT "";
ALTER TABLE profile_showcase_assets_preferences
ADD COLUMN community_id TEXT DEFAULT "";

ALTER TABLE profile_showcase_collectibles_contacts
ADD COLUMN community_id TEXT DEFAULT "";

ALTER TABLE profile_showcase_assets_contacts
ADD COLUMN contract_address TEXT DEFAULT "";
ALTER TABLE profile_showcase_assets_contacts
ADD COLUMN community_id TEXT DEFAULT "";
