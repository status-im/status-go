INSERT INTO keycards (keycard_uid, keycard_name, keycard_locked, key_uid, position)
SELECT
    s.keycard_instance_uid,
    s.name,
    0 AS keycard_locked,
    s.key_uid,
    0 AS position
FROM settings s
WHERE s.keycard_instance_uid IS NOT NULL
  AND s.keycard_instance_uid NOT IN (SELECT keycard_uid FROM keycards);

INSERT INTO keycards_accounts (keycard_uid, account_address)
SELECT
    k.keycard_uid,
    kpa.address
FROM keypairs_accounts kpa
JOIN keycards k ON k.key_uid = kpa.key_uid
WHERE kpa.chat = 0
  AND kpa.key_uid IN (SELECT key_uid FROM settings WHERE keycard_instance_uid IS NOT NULL)
  AND NOT EXISTS (
        SELECT 1
        FROM keycards_accounts ka
        WHERE ka.account_address = kpa.address
    );