-- This migration updates the ens_usernames table to fix entries when the Status domain was set twice
-- eg: test.stateofus.eth.stateofus.eth is changed to test.stateofus.eth

UPDATE ens_usernames
SET username = REPLACE(username, '.stateofus.eth.stateofus.eth', '.stateofus.eth')
WHERE username LIKE '%.stateofus.eth.stateofus.eth';
