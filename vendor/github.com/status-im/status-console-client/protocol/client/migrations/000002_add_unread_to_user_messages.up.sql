ALTER TABLE user_messages
    ADD COLUMN flags INT NOT NULL DEFAULT 0; -- various message flags like read/unread
