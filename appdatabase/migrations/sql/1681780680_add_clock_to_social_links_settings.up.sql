ALTER TABLE social_links_settings ADD COLUMN clock INT DEFAULT 0;
UPDATE social_links_settings SET clock = 1 WHERE link_url IS NOT NULL;
