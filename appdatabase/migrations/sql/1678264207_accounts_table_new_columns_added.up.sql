ALTER TABLE accounts ADD keypair_name TEXT DEFAULT "";
ALTER TABLE accounts ADD last_used_derivation_index INT NOT NULL DEFAULT 0;