ALTER TABLE route_input_parameters ADD COLUMN from_address BLOB NOT NULL AS (unhex(substr(json_extract(route_input_params_json, '$.addrFrom'),3)));
ALTER TABLE route_input_parameters ADD COLUMN to_address BLOB NOT NULL AS (unhex(substr(json_extract(route_input_params_json, '$.addrTo'),3)));

CREATE INDEX IF NOT EXISTS idx_route_input_parameters_per_from_address ON route_input_parameters (from_address);
CREATE INDEX IF NOT EXISTS idx_route_input_parameters_per_to_address ON route_input_parameters (to_address);
