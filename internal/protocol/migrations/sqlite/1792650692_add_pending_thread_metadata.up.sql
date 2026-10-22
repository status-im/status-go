-- Buffers thread rename metadata received before the thread's creation
-- metadata, so the rename can be validated and replayed once the creator is
-- known. Only the latest rename per signer is kept. Rows are bounded per
-- signer and per chat, and expire based on received_at (milliseconds).
CREATE TABLE IF NOT EXISTS pending_thread_metadata (
    chat_id TEXT NOT NULL,
    thread_id TEXT NOT NULL,
    signer_pubkey BLOB NOT NULL,
    name TEXT NOT NULL,
    clock INTEGER NOT NULL,
    received_at INTEGER NOT NULL,
    PRIMARY KEY (chat_id, thread_id, signer_pubkey)
);

CREATE INDEX IF NOT EXISTS idx_pending_thread_metadata_chat_signer ON pending_thread_metadata(chat_id, signer_pubkey);
CREATE INDEX IF NOT EXISTS idx_pending_thread_metadata_received_at ON pending_thread_metadata(received_at);
