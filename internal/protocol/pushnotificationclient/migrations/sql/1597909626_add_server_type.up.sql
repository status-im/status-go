ALTER TABLE push_notification_client_servers ADD COLUMN server_type INT DEFAULT 2;
UPDATE push_notification_client_servers SET server_type = 2;

