-- A startup race let a zero-length reconcile window persist "now" as every
-- topic's history cursor before the cursor-based catch-up ran, permanently
-- skipping the gap since the app was last open. Reset cursors so the next
-- catch-up re-fetches the default sync period and recovers those messages.
UPDATE mailserver_topics SET last_request = 0;
