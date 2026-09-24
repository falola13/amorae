-- +goose Up
-- The moments a couple keeps. A quiet archive, read far more often than it is
-- written to, and never a feed: there is no author, no reaction and no
-- ordering but the day it happened.

CREATE TABLE memories (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    -- The day the moment happened. The composer only ever offers today,
    -- because the thing it is for is saving something as it happens — but
    -- the column is a plain date, so a screen that one day lets you write
    -- down last Saturday needs nothing here to change.
    date       DATE NOT NULL,
    location   TEXT NULL,
    note       TEXT NULL,
    -- Whether a photo is attached (FR-MEM-003). The server owns this: it
    -- becomes true when an upload finishes, never because a client said so,
    -- or a memory could claim a picture that does not exist. Nothing sets it
    -- yet — photos wait on the bucket Q-06 chose.
    has_photo  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every read of this is "this couple's, newest first".
CREATE INDEX memories_couple_idx ON memories (couple_id, date DESC, created_at DESC);

-- +goose Down
DROP TABLE memories;
