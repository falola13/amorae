-- +goose Up
-- The dates a couple keeps: birthdays, anniversaries, the day they met, the
-- day they moved in. One table rather than one per kind (BR-DATE-01) — they
-- differ only in what they are called, and a schema that says otherwise makes
-- every reader of it choose a category before they can write down a day.

CREATE TABLE milestones (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    -- The day it happened, not the next time it comes round. A birthday is
    -- stored as the birth date, and which year's occurrence is being looked
    -- at is worked out when it is asked for — by the client for the list, by
    -- the worker for the reminder. Storing "next year" would need rewriting
    -- every January.
    date       DATE NOT NULL,
    -- A line under the title, in their own words ("Three years together").
    sub        TEXT NULL,
    -- "Remind us every year", as the composer puts it. It is one flag doing
    -- one thing: whether this date comes round for them, or is simply kept.
    -- The list screen reads it the same way — a date with it off belongs to
    -- the story, not to what is coming up.
    reminder   BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX milestones_couple_idx ON milestones (couple_id, date);

-- +goose Down
DROP TABLE milestones;
