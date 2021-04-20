ALTER TABLE app_metrics RENAME TO temp_app_metrics;
CREATE TABLE app_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    proto_id VARCHAR,
    event VARCHAR NOT NULL,
    value TEXT NOT NULL,
    app_version VARCHAR NOT NULL,
    operating_system VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR,
    processed BOOLEAN NOT NULL DEFAULT FALSE);
INSERT INTO app_metrics(event, value, app_version, operating_system, created_at, session_id, processed)
SELECT event, value, app_version, operating_system, created_at, session_id, processed
FROM temp_app_metrics;
DROP TABLE temp_app_metrics;