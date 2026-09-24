-- +goose Up
-- A short note from one partner to the other. Personal, not social: no feed,
-- no likes, no reactions, and nothing here counts them.

CREATE TABLE appreciations (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    -- Who sent it. There is no recipient column: a couple has exactly two
    -- members, so the recipient is the other one (BR-APPR-01, DEC-16).
    from_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date       DATE NOT NULL,
    text       TEXT NOT NULL,
    -- What the undo window is measured from, so it has to be an instant and
    -- not the date. The sender may take a note back within thirty seconds of
    -- this (BR-APPR-02); the client offers five, and the rest absorbs a slow
    -- connection.
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX appreciations_couple_idx ON appreciations (couple_id, created_at DESC);

-- +goose Down
DROP TABLE appreciations;
