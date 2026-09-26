-- +goose Up
-- A history of what was sent, so the app itself has somewhere to show it —
-- notification_sends exists to stop a second send, not to be read back, and
-- has no title or body to show anyone.
CREATE TABLE notification_inbox (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Mirrors notification_sends.kind — 'appreciation', 'goal_crossing', etc.
    kind       TEXT NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    path       TEXT NOT NULL DEFAULT '/',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Null until the person opens their inbox; the read-all endpoint sets
    -- every one of theirs at once, so there is no per-row "mark this read".
    read_at    TIMESTAMPTZ NULL
);

-- The list is always "this person's, newest first"; nothing else is ever
-- queried by.
CREATE INDEX notification_inbox_user_created_idx
    ON notification_inbox (user_id, created_at DESC);

-- The "thinking of you" nudge gets its own switch like every other kind;
-- it had none, so turning it off meant turning everything off.
ALTER TABLE notification_preferences ADD COLUMN nudges BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN IF EXISTS nudges;
DROP TABLE notification_inbox;
