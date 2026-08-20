UPDATE contacts SET contact_request_state = CASE
    WHEN added THEN
      2
  END;

UPDATE contacts SET contact_request_remote_state = CASE
    WHEN has_added_us THEN
      3
  END;
