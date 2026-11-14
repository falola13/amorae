-- +goose Up
-- One line a partner asks the other to pray for, for one day. The day is the
-- couple's own calendar day, decided in Go (never CURRENT_DATE, which is UTC).
-- Once the other partner has prayed for it, it is kept as it was.

CREATE TABLE prayer_requests (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asked_on   DATE NOT NULL,
    text       TEXT NOT NULL,
    prayed_by  UUID REFERENCES users (id) ON DELETE SET NULL,
    prayed_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (couple_id, author_id, asked_on),
    -- Both or neither: "prayed" without who, or who without when, is nonsense.
    CHECK ((prayed_by IS NULL) = (prayed_at IS NULL))
);

-- Whether the other partner wants to hear about it. On by default.
ALTER TABLE notification_preferences
    ADD COLUMN prayer_requests BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN prayer_requests;
DROP TABLE prayer_requests;