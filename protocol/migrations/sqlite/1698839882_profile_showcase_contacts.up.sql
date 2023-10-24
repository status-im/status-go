DROP TABLE profile_showcase_contacts;

CREATE TABLE profile_showcase_contacts (
    contact_id TEXT NOT NULL,
    entry_id TEXT NOT NULL,
    entry_type INT NOT NULL,
    entry_order INT DEFAULT 0,
    PRIMARY KEY (contact_id, entry_id)
);
