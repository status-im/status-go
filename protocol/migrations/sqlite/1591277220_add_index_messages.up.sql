DROP INDEX idx_search_by_chat_id;

CREATE INDEX idx_search_by_local_chat_id_sort_on_cursor ON user_messages (local_chat_id ASC, substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id DESC);
