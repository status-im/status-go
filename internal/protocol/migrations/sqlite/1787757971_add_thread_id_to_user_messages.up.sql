ALTER TABLE user_messages ADD COLUMN thread_id TEXT;

CREATE INDEX user_messages_thread_id ON user_messages(thread_id);

CREATE INDEX user_messages_local_chat_id_thread_id_clock_value_idx ON user_messages(local_chat_id, thread_id, clock_value);
