CREATE TABLE envelopes (id BYTEA NOT NULL UNIQUE, data BYTEA NOT NULL, topic BYTEA NOT NULL, bloom BIT(512) NOT NULL);

CREATE INDEX id_bloom_idx ON envelopes (id DESC, bloom);
CREATE INDEX id_topic_idx ON envelopes (id DESC, topic);
