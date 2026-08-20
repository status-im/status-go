
-- Create tables for storing social links in the prodile showcase
CREATE TABLE profile_showcase_social_links_preferences (
    url VARCHAR NOT NULL CHECK (length(trim(url)) > 0),
    text VARCHAR NOT NULL CHECK (length(trim(text)) > 0),
    visibility INT NOT NULL DEFAULT 0,
    sort_order INT DEFAULT 0,
    PRIMARY KEY (text, url)
);

CREATE TABLE profile_showcase_social_links_contacts (
    url VARCHAR NOT NULL CHECK (length(trim(url)) > 0),
    text VARCHAR NOT NULL CHECK (length(trim(text)) > 0),
    sort_order INT DEFAULT 0,
    contact_id TEXT NOT NULL,
    PRIMARY KEY (contact_id, text, url)
);
CREATE INDEX profile_showcase_social_links_contact_id ON profile_showcase_social_links_contacts (contact_id);

-- Copy existing social links to a new table
INSERT INTO profile_showcase_social_links_preferences (text, url, sort_order)
    SELECT text, url, position
    FROM profile_social_links;
