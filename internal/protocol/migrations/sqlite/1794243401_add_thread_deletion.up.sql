ALTER TABLE threads ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN deleted_clock INTEGER NOT NULL DEFAULT 0;
ALTER TABLE threads ADD COLUMN deleted_by TEXT NOT NULL DEFAULT '';
ALTER TABLE threads ADD COLUMN deleted_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pending_thread_metadata ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pending_thread_metadata ADD COLUMN deleted_at INTEGER NOT NULL DEFAULT 0;

-- Enforce tombstones for every write path, including edits and backup restores.
CREATE TRIGGER deleted_thread_message_insert AFTER INSERT ON user_messages
WHEN NEW.thread_id <> '' AND EXISTS (SELECT 1 FROM threads WHERE thread_id = NEW.thread_id AND chat_id = NEW.local_chat_id AND deleted)
BEGIN
 UPDATE user_messages SET deleted = TRUE, seen = TRUE, deleted_by = (SELECT deleted_by FROM threads WHERE thread_id = NEW.thread_id AND chat_id = NEW.local_chat_id) WHERE id = NEW.id;
END;
CREATE TRIGGER deleted_thread_message_update AFTER UPDATE OF deleted, thread_id ON user_messages
WHEN NEW.deleted = FALSE AND NEW.thread_id <> '' AND EXISTS (SELECT 1 FROM threads WHERE thread_id = NEW.thread_id AND chat_id = NEW.local_chat_id AND deleted)
BEGIN
 UPDATE user_messages SET deleted = TRUE, seen = TRUE, deleted_by = (SELECT deleted_by FROM threads WHERE thread_id = NEW.thread_id AND chat_id = NEW.local_chat_id) WHERE id = NEW.id;
END;
