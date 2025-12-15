DROP TABLE IF EXISTS app_metrics;
ALTER TABLE settings DROP COLUMN anon_metrics_should_send;
ALTER TABLE shhext_config DROP COLUMN anon_metrics_server_enabled;
ALTER TABLE shhext_config DROP COLUMN anon_metrics_send_id;
ALTER TABLE shhext_config DROP COLUMN anon_metrics_server_postgres_uri;