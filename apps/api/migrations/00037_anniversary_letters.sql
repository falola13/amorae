-- +goose Up
-- A letter one partner writes the other for their anniversary. Written in
-- the days before, sealed until the day itself: the server never hands the
-- other partner its text until the couple's own calendar reaches opens_on.
-- After that it stays, one per person per year, part of their story.

CREATE TABLE anniversary_letters (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    -- No recipient column, as with appreciations: it is the other one.
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- The anniversary it is for, in the couple's own calendar.
    opens_on   DATE NOT NULL,
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (couple_id, author_id, opens_on)
);

CREATE INDEX anniversary_letters_couple_idx ON anniversary_letters (couple_id, opens_on DESC);

-- +goose Down
DROP TABLE anniversary_letters;
