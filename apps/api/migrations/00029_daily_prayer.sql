-- +goose Up
-- Prayer moves from "once and done" to "every day": a point can now say
-- which days it's for, either partner can write and publish the week, and a
-- completion is recorded per day rather than forever.

-- Empty (the default, 127 = all seven bits) means every day; a point never
-- has to say so. Bit n is weekday n, Sunday=0..Saturday=6, matching Postgres's
-- own EXTRACT(DOW ...) and the app's time.Weekday.
ALTER TABLE prayer_points
    ADD COLUMN weekdays SMALLINT NOT NULL DEFAULT 127 CHECK (weekdays BETWEEN 1 AND 127);

-- Either partner may publish now; this is who did, so KindWeekPublished can
-- be addressed to whoever didn't, instead of always assuming the setter.
ALTER TABLE prayer_weeks
    ADD COLUMN published_by UUID NULL REFERENCES users (id) ON DELETE SET NULL;

-- prayer_completions moves from "prayed, ever" to "prayed, on this day".
ALTER TABLE prayer_completions ADD COLUMN prayed_on DATE;

-- Backfill: a completion recorded before today existed happened on the
-- couple-local date it was completed, so that's the day it counts as now.
UPDATE prayer_completions c
SET prayed_on = (c.completed_at AT TIME ZONE co.timezone)::date
FROM prayer_points p
JOIN prayer_weeks w ON w.id = p.week_id
JOIN couples co ON co.id = w.couple_id
WHERE p.id = c.point_id;

ALTER TABLE prayer_completions ALTER COLUMN prayed_on SET NOT NULL;

-- "Marking it twice changes nothing" now means "on the same day"; a
-- different day is a different prayer, not the same one repeated.
ALTER TABLE prayer_completions DROP CONSTRAINT prayer_completions_pkey;
ALTER TABLE prayer_completions ADD PRIMARY KEY (point_id, user_id, prayed_on);

-- The reminder and both-prayed checks both ask "what happened today",
-- across points/users, which the PK (point_id-first) doesn't serve directly.
CREATE INDEX prayer_completions_prayed_on_idx ON prayer_completions (prayed_on);

-- +goose Down
-- Down cannot bring back "prayed forever" from "prayed on this day" without
-- losing the day-by-day history — that history is the whole point of the Up.
-- This keeps one row per (point_id, user_id), the earliest day it happened,
-- and accepts the rest as gone.
DROP INDEX IF EXISTS prayer_completions_prayed_on_idx;

DELETE FROM prayer_completions a
WHERE EXISTS (
    SELECT 1 FROM prayer_completions b
    WHERE b.point_id = a.point_id AND b.user_id = a.user_id AND b.prayed_on < a.prayed_on
);

ALTER TABLE prayer_completions DROP CONSTRAINT prayer_completions_pkey;
ALTER TABLE prayer_completions ADD PRIMARY KEY (point_id, user_id);
ALTER TABLE prayer_completions DROP COLUMN prayed_on;

ALTER TABLE prayer_weeks DROP COLUMN published_by;
ALTER TABLE prayer_points DROP COLUMN weekdays;
