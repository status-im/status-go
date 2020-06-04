DROP INDEX idx_search_by_local_chat_id_sort_on_cursor;

CREATE INDEX idx_search_by_chat_id ON  user_messages(
    substr('0000000000000000000000000000000000000000000000000000000000000000' || clock_value, -64, 64) || id, chat_id, hide
);
