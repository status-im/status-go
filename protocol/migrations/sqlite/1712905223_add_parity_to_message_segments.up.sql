ALTER TABLE message_segments RENAME TO old_message_segments;

CREATE TABLE message_segments (
    hash BLOB NOT NULL,
    segment_index INTEGER NOT NULL,
    segments_count INTEGER NOT NULL,
    payload BLOB NOT NULL,
    sig_pub_key BLOB NOT NULL,
    timestamp INTEGER NOT NULL,
    parity_segment_index INTEGER NOT NULL,
    parity_segments_count INTEGER NOT NULL,
    PRIMARY KEY (hash, sig_pub_key, segment_index, segments_count, parity_segment_index, parity_segments_count) ON CONFLICT REPLACE
);

INSERT INTO message_segments (hash, segment_index, segments_count, payload, sig_pub_key, timestamp, parity_segment_index, parity_segments_count)
SELECT hash, segment_index, segments_count, payload, sig_pub_key, timestamp, 0, 0
FROM old_message_segments;

DROP TABLE old_message_segments;
