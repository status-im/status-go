-- Marks threads created before their creation metadata arrived (for example,
-- from an incoming reply). Unlike legacy creator-unknown threads, these may be
-- claimed by the first valid creation metadata.
ALTER TABLE threads ADD COLUMN creator_pending BOOLEAN NOT NULL DEFAULT FALSE;
