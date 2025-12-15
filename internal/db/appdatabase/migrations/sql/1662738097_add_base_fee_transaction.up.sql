ALTER TABLE transfers ADD COLUMN base_gas_fee TEXT NOT NULL DEFAULT "";
UPDATE transfers SET base_gas_fee = "";
