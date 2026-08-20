UPDATE user_messages
SET seen = 1
WHERE
    discord_message_id IS NOT NULL
    AND discord_message_id != ''
