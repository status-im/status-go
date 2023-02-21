CREATE TABLE  IF NOT EXISTS activity_center_states (
  has_seen BOOLEAN
);

INSERT INTO activity_center_states SELECT 1  WHERE NOT EXISTS(SELECT 1 FROM activity_center_states);

