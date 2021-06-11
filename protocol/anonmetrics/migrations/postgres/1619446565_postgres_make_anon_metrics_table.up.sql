CREATE TABLE app_metrics (
    id SERIAL PRIMARY KEY,

    /* Incoming metric data */
    message_id VARCHAR UNIQUE NOT NULL,
    event VARCHAR NOT NULL,
    value JSON NOT NULL,
    app_version VARCHAR NOT NULL,
    operating_system VARCHAR NOT NULL,
    session_id VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL,

    /* Row metadata */
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);