-- +goose Up
-- A prayer that was answered. The app has always asked the two of them to
-- pray and then never asked what happened, so a week closed into history and
-- that was the end of it. This is the other half: the thing a prayer journal
-- is actually kept for.
--
-- It lives on the point rather than in a join table because, unlike praying,
-- answering is not per-person: a prayer is answered for the couple, once.
-- `answered_by` records who noticed, which is worth keeping and is what the
-- notification to the other partner is addressed from.
ALTER TABLE prayer_points ADD COLUMN answered_at TIMESTAMPTZ NULL;
ALTER TABLE prayer_points ADD COLUMN answered_by UUID NULL REFERENCES users (id);
ALTER TABLE prayer_points ADD COLUMN answer_note TEXT NOT NULL DEFAULT '';

-- The read-back screen asks for every answered point a couple has, newest
-- first, across every week. Partial, because the answered ones are the small
-- minority this index exists to find.
CREATE INDEX prayer_points_answered_idx
    ON prayer_points (answered_at DESC)
    WHERE answered_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS prayer_points_answered_idx;
ALTER TABLE prayer_points DROP COLUMN answer_note;
ALTER TABLE prayer_points DROP COLUMN answered_by;
ALTER TABLE prayer_points DROP COLUMN answered_at;
