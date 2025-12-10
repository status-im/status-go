ALTER TABLE networks ADD COLUMN chain_color VARCHAR NOT NULL DEFAULT "";
UPDATE networks SET chain_color = "";
ALTER TABLE networks ADD COLUMN short_name VARCHAR NOT NULL DEFAULT "";
UPDATE networks SET short_name = "";
