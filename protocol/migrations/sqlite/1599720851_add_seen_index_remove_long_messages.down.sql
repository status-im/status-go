DROP INDEX seen_local_chat_id_idx;
UPDATE user_messages SET hide = 0 WHERE length(text) > 4096;
