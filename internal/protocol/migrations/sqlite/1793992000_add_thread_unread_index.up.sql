CREATE INDEX user_messages_unread_threads
ON user_messages(local_chat_id, thread_id, mentioned, replied)
WHERE NOT seen AND NOT hide AND NOT deleted AND NOT deleted_for_me AND thread_id != '';
