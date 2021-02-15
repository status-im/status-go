CREATE TABLE IF NOT EXISTS metrics (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       event VARCHAR NOT NULL,
       value TEXT NOT NULL,
       app_version VARCHAR NOT NULL,
       operating_system VARCHAR NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
