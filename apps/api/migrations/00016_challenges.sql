-- +goose Up
-- Short, guided, multi-day experiences. Optional, and never framed as a
-- streak somebody can break.

CREATE TABLE challenges (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    -- Which curated set this came from, so a challenge can be recognised
    -- later even if its prompts are reworded.
    template   TEXT NOT NULL,
    title      TEXT NOT NULL,
    started_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One at a time. "Current" has to mean something, and two half-finished
-- challenges is not a thing anyone wants to come back to.
CREATE UNIQUE INDEX challenges_one_live_per_couple ON challenges (couple_id);

CREATE TABLE challenge_days (
    id           UUID PRIMARY KEY,
    challenge_id UUID NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    -- 1-based: people count days from one.
    n            SMALLINT NOT NULL CHECK (n >= 1),
    prompt       TEXT NOT NULL,
    CONSTRAINT challenge_days_number_key UNIQUE (challenge_id, n)
);

CREATE TYPE challenge_mark AS ENUM ('done', 'skipped');

-- One row per partner per day (DEC-30). A single shared flag would let one
-- partner tick a day for both, which misreports what each of them did — and
-- what each of them kept up is the whole point. Same shape as
-- prayer_completions, for the same reason.
--
-- Marking a day skipped after marking it done replaces the row rather than
-- adding one: a day has one state per person, not a history of opinions.
CREATE TABLE challenge_progress (
    day_id    UUID NOT NULL REFERENCES challenge_days (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mark      challenge_mark NOT NULL,
    marked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (day_id, user_id)
);

CREATE INDEX challenge_progress_user_idx ON challenge_progress (user_id);

-- +goose Down
DROP TABLE challenge_progress;
DROP TYPE challenge_mark;
DROP TABLE challenge_days;
DROP TABLE challenges;
