-- Tracks the time (milliseconds) of the latest new message in a thread,
-- used to auto-archive inactive threads. Existing threads are backfilled from
-- the latest message timestamp. When no timestamp is available, use the migration
-- time since legacy threads have no recorded creation time. Future timestamps
-- are capped at migration time.
ALTER TABLE threads ADD COLUMN last_activity_at INTEGER NOT NULL DEFAULT 0;

UPDATE threads SET last_activity_at = MIN(COALESCE(NULLIF((
    SELECT MAX(user_messages.timestamp)
    FROM user_messages
    WHERE (user_messages.id = threads.parent_message_id
        OR (user_messages.local_chat_id = threads.chat_id AND user_messages.thread_id = threads.thread_id))
        AND NOT user_messages.deleted
        AND NOT user_messages.deleted_for_me
), 0), CAST(strftime('%s', 'now') AS INTEGER) * 1000),
    CAST(strftime('%s', 'now') AS INTEGER) * 1000);

CREATE INDEX IF NOT EXISTS threads_last_activity_at_idx ON threads(last_activity_at);
