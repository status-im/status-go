ALTER TABLE message_segments
ADD COLUMN timestamp INTEGER DEFAULT 0;

ALTER TABLE message_segments_completed
ADD COLUMN timestamp INTEGER DEFAULT 0;

CREATE INDEX idx_message_segments_timestamp ON message_segments(timestamp);
CREATE INDEX idx_message_segments_completed_timestamp ON message_segments_completed(timestamp);
