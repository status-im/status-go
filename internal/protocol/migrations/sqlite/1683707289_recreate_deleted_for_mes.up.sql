CREATE TABLE IF NOT EXISTS user_messages_deleted_for_mes_bk (
    clock      INTEGER NOT NULL,
    message_id VARCHAR NOT NULL,
    PRIMARY KEY (message_id)
);

INSERT OR REPLACE INTO user_messages_deleted_for_mes_bk SELECT clock, message_id FROM user_messages_deleted_for_mes;

DROP TABLE user_messages_deleted_for_mes;

ALTER TABLE user_messages_deleted_for_mes_bk RENAME TO user_messages_deleted_for_mes;
