-- Whether the user follows a thread. followed_clock is 0 until the user
-- explicitly follows or unfollows (including by replying), so automatic
-- follows never override an explicit choice.
ALTER TABLE threads ADD COLUMN followed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN followed_clock INTEGER NOT NULL DEFAULT 0;
