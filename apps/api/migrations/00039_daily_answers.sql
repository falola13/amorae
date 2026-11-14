-- +goose Up
-- One person's answer to the couple's question of the day. The question is
-- copied in as asked, so editing the list in code never rewrites what was
-- answered. The day is the couple's own calendar day, decided in Go. The
-- server holds each answer back from the other partner until they have
-- answered that day too.

CREATE TABLE daily_answers (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asked_on   DATE NOT NULL,
    question   TEXT NOT NULL,
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (couple_id, author_id, asked_on)
);

CREATE INDEX daily_answers_couple_idx ON daily_answers (couple_id, asked_on DESC);

-- Whether the other partner answering is announced. On by default.
ALTER TABLE notification_preferences
    ADD COLUMN daily_question BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN daily_question;
DROP TABLE daily_answers;
