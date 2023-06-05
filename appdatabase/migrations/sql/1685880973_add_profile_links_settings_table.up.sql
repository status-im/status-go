-- Create new table where we will store profile social links,
-- after discussion https://github.com/status-im/status-go/pull/3552#issuecomment-1573817044
-- we decided to replace the entire db table content on every change (because of that position column is not needed)
CREATE TABLE IF NOT EXISTS profile_social_links (
  text VARCHAR NOT NULL CHECK (length(trim(text)) > 0),
  url VARCHAR NOT NULL CHECK (length(trim(url)) > 0),
  position INT DEFAULT 0,
  PRIMARY KEY (text, url)
);

-- Create new column to keep clock of the last change of profile social links (since content is replaced all row share the same clock value)
ALTER TABLE settings_sync_clock ADD COLUMN social_links INTEGER NOT NULL DEFAULT 0;

-- Create temp table to generate a position for rows
CREATE TEMPORARY TABLE temp_social_links_settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  text VARCHAR NOT NULL,
  url VARCHAR NOT NULL
);

-- Copy to temp table
INSERT INTO temp_social_links_settings (text, url)
  SELECT link_text, link_url
  FROM social_links_settings
  WHERE link_url != "";

-- Insert into `profile_social_links` table
INSERT INTO profile_social_links
  SELECT text, url, id
  FROM temp_social_links_settings;

-- From some reason the following sql doesn't work through status-go migrations, although the query is correct.
-- -- Set the clock for profile social links
-- UPDATE settings_sync_clock
--   SET
--     social_links = (SELECT clock
--     FROM social_links_settings
--     WHERE link_url != ""
--     LIMIT 1)
--   WHERE synthetic_id = "id";

-- Drop temp table
DROP TABLE temp_social_links_settings;

-- Drop old table
DROP TABLE social_links_settings;