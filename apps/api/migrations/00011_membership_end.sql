-- +goose Up
-- Q-24: ending a couple must not cost someone a month of the app.
--
-- The membership row is what keeps an ended couple readable, and
-- UNIQUE (user_id) made holding one the same thing as being in a couple — so
-- a former partner could not start a new one until the old was purged.
-- Marking the membership ended separates the two ideas: an ended membership
-- goes on carrying read access, and only live ones are exclusive.
ALTER TABLE couple_members ADD COLUMN ended_at TIMESTAMPTZ NULL;

-- Couples that ended before this migration: their members end with them.
UPDATE couple_members m
SET ended_at = c.dissolved_at
FROM couples c
WHERE c.id = m.couple_id AND c.dissolved_at IS NOT NULL;

ALTER TABLE couple_members DROP CONSTRAINT couple_members_user_id_key;

-- One live couple per person, exactly as before. The name says "live", so a
-- unique violation on it still means "already paired".
CREATE UNIQUE INDEX couple_members_live_user_key ON couple_members (user_id) WHERE ended_at IS NULL;

-- "What did I used to be part of": few rows per person, and the purge keeps
-- it that way, but the archive lookup shouldn't scan the table.
CREATE INDEX couple_members_ended_user_idx ON couple_members (user_id, ended_at)
    WHERE ended_at IS NOT NULL;

-- +goose Down
-- The old constraint allows one membership per person, so rolling back has to
-- drop the ended ones. That loses read access to couples that have ended but
-- not yet been purged; there is no way to keep both and go back.
DROP INDEX couple_members_ended_user_idx;
DROP INDEX couple_members_live_user_key;
DELETE FROM couple_members WHERE ended_at IS NOT NULL;
ALTER TABLE couple_members ADD CONSTRAINT couple_members_user_id_key UNIQUE (user_id);
ALTER TABLE couple_members DROP COLUMN ended_at;
