-- +goose Up
-- An event can carry several reminders, and once it is over it can say how it
-- went. Reminders were a single free-text phrase; they become a list of
-- phrases (or "at 16:00"-style clock times), kept in the order they were
-- given. The old single value is carried over as a list of one.
ALTER TABLE events
    ADD COLUMN reminders TEXT[] NOT NULL DEFAULT '{}';
UPDATE events SET reminders = ARRAY[reminder] WHERE COALESCE(reminder, '') <> '';
ALTER TABLE events DROP COLUMN reminder;

-- What happened to it, when the couple has said: done ("it happened") or
-- didnt_happen. The two are never both true; neither is "not said yet", which
-- is what every event starts as.
ALTER TABLE events
    ADD COLUMN didnt_happen BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE events
    ADD CONSTRAINT events_outcome_exclusive CHECK (NOT (done AND didnt_happen));

-- +goose Down
ALTER TABLE events DROP CONSTRAINT events_outcome_exclusive;
ALTER TABLE events DROP COLUMN didnt_happen;
-- Only the first reminder survives going back: the old column held one.
ALTER TABLE events ADD COLUMN reminder TEXT NULL;
UPDATE events SET reminder = reminders[1];
ALTER TABLE events DROP COLUMN reminders;
