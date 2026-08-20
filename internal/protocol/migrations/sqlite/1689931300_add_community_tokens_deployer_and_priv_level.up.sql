ALTER TABLE community_tokens ADD COLUMN deployer TEXT NOT NULL DEFAULT "";
ALTER TABLE community_tokens ADD COLUMN privileges_level INT NOT NULL DEFAULT 2;
