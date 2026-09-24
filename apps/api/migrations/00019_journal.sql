-- +goose Up
-- The couple's shared journal. Private to the two of them, and unlike a
-- memory or an event it has an author on purpose: "you wrote this on Tuesday"
-- is part of reading it back, and the screen says who beside every entry.

-- The set the client offers, spelled the way it sends them. An enum rather
-- than free text because five choices in a picker is a closed set, and one
-- that drifts is one the screen cannot group by.
CREATE TYPE journal_tag AS ENUM ('Gratitude', 'Reflection', 'Memory', 'Appreciation', 'Plans');

CREATE TABLE journal_entries (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    -- Deleting an account takes its entries with it: what somebody wrote is
    -- authored personal data and follows the account (DEC-25).
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date       DATE NOT NULL,
    tag        journal_tag NOT NULL,
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every read is "this couple's, newest first".
CREATE INDEX journal_entries_couple_idx ON journal_entries (couple_id, date DESC, created_at DESC);

-- +goose Down
DROP TABLE journal_entries;
DROP TYPE journal_tag;
