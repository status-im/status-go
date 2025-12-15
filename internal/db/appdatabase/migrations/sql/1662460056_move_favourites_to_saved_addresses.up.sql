ALTER TABLE saved_addresses ADD COLUMN favourite BOOLEAN NOT NULL DEFAULT FALSE;

INSERT OR REPLACE INTO saved_addresses(address, name, favourite, network_id) SELECT address, name, "TRUE", "1" FROM favourites;

DROP TABLE favourites;