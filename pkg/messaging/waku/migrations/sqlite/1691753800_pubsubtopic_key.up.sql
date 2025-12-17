CREATE TABLE IF NOT EXISTS pubsubtopic_signing_key (
    topic VARCHAR NOT NULL,
    priv_key BLOB NULL,
    pub_key BLOB NOT NULL,
    PRIMARY KEY (topic)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS mailserver_topics_new (
    topic VARCHAR NOT NULL DEFAULT "",
    pubsub_topic VARCHAR NOT NULL DEFAULT "/waku/2/default-waku/proto",
    chat_ids VARCHAR,
    last_request INTEGER DEFAULT 1,
    discovery BOOLEAN DEFAULT FALSE,
    negotiated BOOLEAN DEFAULT FALSE,
    PRIMARY KEY(topic, pubsub_topic)
) WITHOUT ROWID;

INSERT INTO mailserver_topics_new 
SELECT topic, "/waku/2/default-waku/proto", chat_ids, last_request, discovery, negotiated
FROM mailserver_topics;

DROP TABLE mailserver_topics;

ALTER TABLE mailserver_topics_new RENAME TO mailserver_topics;