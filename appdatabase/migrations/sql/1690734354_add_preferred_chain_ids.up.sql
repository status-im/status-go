ALTER TABLE keypairs_accounts ADD COLUMN prod_preferred_chain_ids VARCHAR NOT NULL DEFAULT "";
ALTER TABLE keypairs_accounts ADD COLUMN test_preferred_chain_ids VARCHAR NOT NULL DEFAULT "";