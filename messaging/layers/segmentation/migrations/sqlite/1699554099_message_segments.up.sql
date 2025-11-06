CREATE TABLE IF NOT EXISTS message_segments (
    hash BLOB NOT NULL,
    segment_index INTEGER NOT NULL,
    segments_count INTEGER NOT NULL,
    payload BLOB NOT NULL,
    sig_pub_key BLOB NOT NULL,
    PRIMARY KEY (hash, sig_pub_key, segment_index) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS message_segments_completed (
    hash BLOB NOT NULL,
    sig_pub_key BLOB NOT NULL,
    PRIMARY KEY (hash, sig_pub_key)
);
