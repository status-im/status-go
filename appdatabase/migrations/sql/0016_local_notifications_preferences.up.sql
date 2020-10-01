CREATE TABLE local_notifications_preferences (
    service VARCHAR NOT NULL,
    event VARCHAR,
    identifier VARCHAR,
    enabled BOOLEAN DEFAULT false,
    PRIMARY KEY(service,event,identifier)
) WITHOUT ROWID;