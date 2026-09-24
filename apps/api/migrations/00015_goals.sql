-- +goose Up
-- Goals the two of them are working toward. One shared total, and a log of
-- who put in what — the log is so the couple can see their own history, not
-- so the total can be split between them (BR-GOAL-01).

CREATE TYPE goal_unit AS ENUM ('naira', 'count');

CREATE TABLE goals (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    -- Why it matters to them. Optional, and worth more than the number.
    why        TEXT NULL,
    -- Whole units: naira, or one of whatever is being counted. Integers
    -- rather than a float, because a total that is the sum of a hundred
    -- entries must still be exactly right — and whole naira rather than
    -- kobo, because that is the number the app shows and nobody is saving
    -- toward a target with kobo in it.
    target     BIGINT NOT NULL CHECK (target > 0),
    unit       goal_unit NOT NULL,
    -- What one counts, when counting ("chapters", "walks"). Free text.
    unit_label TEXT NULL,
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    done       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT goals_dates_in_order CHECK (end_date >= start_date)
);

CREATE INDEX goals_couple_idx ON goals (couple_id, created_at DESC);

-- There is no current_value column on purpose. The running total is the sum
-- of these rows, computed on read: a stored counter and a log that disagree
-- is a bug waiting to happen, and correcting one entry would mean correcting
-- two places.
CREATE TABLE goal_progress (
    id       UUID PRIMARY KEY,
    goal_id  UUID NOT NULL REFERENCES goals (id) ON DELETE CASCADE,
    -- Who logged it. For the log only (BR-GOAL-01) — the total is the
    -- couple's, not two totals side by side.
    user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Negative is allowed: a correction is how you fix a number you entered
    -- wrong, and deleting the entry would lose the fact that it happened.
    amount   BIGINT NOT NULL,
    date     DATE NOT NULL,
    logged_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX goal_progress_goal_idx ON goal_progress (goal_id, date);

-- +goose Down
DROP TABLE goal_progress;
DROP TABLE goals;
DROP TYPE goal_unit;
