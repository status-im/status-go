-- Between 2.37 and the switch to the emoji font, editing a wallet account's emoji on desktop stored the
-- Twemoji <img> markup the picker emitted instead of the emoji itself, e.g.:
--   <img class="emoji" draggable="false" alt="🐟" src="qrc:/assets/twemoji/svg/1f41f.svg" width="18" height="18" style="vertical-align: top">
-- Replace such values with the content of their alt attribute, which is the emoji the user picked.
-- https://github.com/status-im/status-app/issues/22908
UPDATE keypairs_accounts
SET emoji = substr(
        emoji,
        instr(emoji, 'alt="') + 5,
        instr(substr(emoji, instr(emoji, 'alt="') + 5), '"') - 1
    )
WHERE emoji LIKE '%<img%'
  AND emoji LIKE '%/twemoji/%'
  AND instr(emoji, 'alt="') > 0
  AND instr(substr(emoji, instr(emoji, 'alt="') + 5), '"') > 1;
