ALTER TABLE chats ADD COLUMN first_unviewed_message_id VARCHAR;

UPDATE chats
SET
    first_unviewed_message_id = (
        SELECT id
        FROM user_messages
        WHERE
            local_chat_id = chats.id
            AND NOT(seen)
            AND NOT(hide)
            AND NOT(deleted)
            AND NOT(deleted_for_me)
        ORDER BY
            clock_value ASC
        LIMIT 1
    );
