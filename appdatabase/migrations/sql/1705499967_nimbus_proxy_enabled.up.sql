ALTER TABLE node_config ADD COLUMN nimbus_proxy_enabled DEFAULT FALSE;
ALTER TABLE node_config ADD COLUMN nimbus_proxy_trusted_block_root VARCHAR NOT NULL DEFAULT "";
