CREATE TABLE centralizedmetrics_metrics (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    event_value BLOB NOT NULL,
    timestamp INTEGER NOT NULL,
    platform VARCHAR NOT NULL,
    app_version VARCHAR NOT NULL
);

CREATE TABLE centralizedmetrics_uuid (
    uuid TEXT PRIMARY KEY,
    enabled BOOL DEFAULT FALSE,
    user_confirmed BOOL DEFAULT FALSE
);
