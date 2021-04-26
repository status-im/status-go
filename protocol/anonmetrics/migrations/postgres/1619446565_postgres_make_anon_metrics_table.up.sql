CREATE TABLE app_metrics (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR,
    event VARCHAR NOT NULL,
    value JSON NOT NULL,
    app_version VARCHAR NOT NULL,
    operating_system VARCHAR NOT NULL,
    session_id VARCHAR,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);