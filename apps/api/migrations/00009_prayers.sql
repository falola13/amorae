-- +goose Up
-- The weekly prayer cycle. Three ideas: a week belongs to a couple and has one
-- setter, a week holds ordered points, and each partner completes points for
-- themselves.

CREATE TYPE prayer_week_status AS ENUM ('draft', 'published');

CREATE TABLE prayer_weeks (
    id             UUID PRIMARY KEY,
    couple_id      UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    -- The local Sunday in the COUPLE's timezone, as a plain date: "which week"
    -- is a calendar fact, not an instant, and must not shift with the server's
    -- clock or a partner's travel.
    week_start     DATE NOT NULL,
    -- Frozen when the week is created, never recomputed on read, so the turn
    -- can't move retroactively if membership data changes.
    setter_user_id UUID NOT NULL REFERENCES users (id),
    status         prayer_week_status NOT NULL DEFAULT 'draft',
    published_at   TIMESTAMPTZ NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- What makes the scheduler safe to run twice, or twice at once: a second
    -- insert for the same week loses instead of duplicating.
    CONSTRAINT prayer_weeks_couple_week_key UNIQUE (couple_id, week_start)
);

-- History is "this couple's weeks, newest first".
CREATE INDEX prayer_weeks_couple_start_idx ON prayer_weeks (couple_id, week_start DESC);

CREATE TABLE prayer_points (
    id         UUID PRIMARY KEY,
    week_id    UUID NOT NULL REFERENCES prayer_weeks (id) ON DELETE CASCADE,
    -- 0-based; the order the setter arranged them in.
    position   SMALLINT NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    scripture  TEXT NULL,
    verse      TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT prayer_points_week_position_key UNIQUE (week_id, position)
);

CREATE INDEX prayer_points_week_idx ON prayer_points (week_id);

-- One row per partner per point. The primary key is the "marking it twice
-- changes nothing" rule, in the schema rather than in application code — which
-- is also what makes the offline queue safe to replay.
CREATE TABLE prayer_completions (
    point_id     UUID NOT NULL REFERENCES prayer_points (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (point_id, user_id)
);

CREATE INDEX prayer_completions_user_idx ON prayer_completions (user_id);

-- A reflection is one partner's own words about the week, and both can see
-- both, so it is keyed per person rather than per week.
CREATE TABLE prayer_reflections (
    week_id    UUID NOT NULL REFERENCES prayer_weeks (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (week_id, user_id)
);

-- +goose Down
DROP TABLE prayer_reflections;
DROP TABLE prayer_completions;
DROP TABLE prayer_points;
DROP TABLE prayer_weeks;
DROP TYPE prayer_week_status;
