ALTER TABLE contacts ADD COLUMN added BOOLEAN DEFAULT FALSE;
ALTER TABLE contacts ADD COLUMN has_added_us BOOLEAN DEFAULT FALSE;
ALTER TABLE contacts ADD COLUMN blocked BOOLEAN DEFAULT FALSE;

CREATE INDEX tmp_contacts ON contacts(hex(system_tags));

UPDATE contacts SET added = 1 WHERE hex(system_tags) LIKE '%6164646564%';
UPDATE contacts SET has_added_us = 1 WHERE hex(system_tags) LIKE '%72656365%';
UPDATE contacts SET blocked = 1 WHERE hex(system_tags) LIKE '%626c6f636b6564%';
CREATE INDEX contacts_added on contacts(added);
CREATE INDEX contacts_has_added_us on contacts(has_added_us);
CREATE INDEX contacts_blocked on contacts(blocked);
CREATE INDEX contacts_local_nickname on contacts(local_nickname);
CREATE INDEX tmp_contacts_delete ON contacts(added, blocked, local_nickname);

DELETE FROM contacts
WHERE NOT(added) AND NOT(blocked) AND NOT(has_added_us) AND (local_nickname == "" OR local_nickname IS NULL)
      AND NOT EXISTS (SELECT 1 FROM chat_identity_contacts i WHERE id = i.contact_id)
      AND NOT EXISTS (SELECT 1 FROM ens_verification_records v  WHERE id = v.public_key);

/*
would be cool to remove these columns

ALTER TABLE contacts DROP COLUMN name;
ALTER TABLE contacts DROP COLUMN ens_verified;
ALTER TABLE contacts DROP COLUMN ens_verified_at;
ALTER TABLE contacts DROP COLUMN device_info;
ALTER TABLE contacts DROP COLUMN system_tags;
ALTER TABLE contacts DROP COLUMN tribute_to_talk;
ALTER TABLE contacts DROP COLUMN last_ens_clock_value;
ALTER TABLE contacts DROP COLUMN photo;
*/

DROP INDEX tmp_contacts_delete;
DROP INDEX tmp_contacts;
