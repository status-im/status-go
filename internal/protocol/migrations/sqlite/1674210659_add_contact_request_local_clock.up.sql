
ALTER TABLE contacts ADD COLUMN contact_request_local_clock INT;
ALTER TABLE contacts ADD COLUMN contact_request_remote_clock INT;
ALTER TABLE contacts ADD COLUMN contact_request_remote_state INT;

-- Broken migration, leaving for posterity and eternal embarrassment
-- but hey, on the bright side, this is valid sql
UPDATE contacts SET contact_request_state = CASE
    WHEN added THEN
      contact_request_state = 2
  END;

UPDATE contacts SET contact_request_local_clock = last_updated_locally;

-- Broken migration, leaving for posterity and eternal embarrassment
UPDATE contacts SET contact_request_remote_state = CASE
    WHEN has_added_us THEN
      contact_request_state = 3
  END;
