CREATE TABLE IF NOT EXISTS mailserver_topics (
    topic VARCHAR PRIMARY KEY,
    chat_ids VARCHAR,
    last_request INTEGER DEFAULT 1,
    discovery BOOLEAN DEFAULT FALSE,
    negotiated BOOLEAN DEFAULT FALSE
) WITHOUT ROWID;
