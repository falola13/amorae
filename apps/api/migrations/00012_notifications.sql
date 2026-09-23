-- +goose Up
-- Notifications are per person, never per couple (FR-NOTF-001): partners
-- choose their own, and one of them turning something off says nothing about
-- the other.

CREATE TABLE notification_preferences (
    user_id         UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    new_week        BOOLEAN NOT NULL DEFAULT true,
    prayer_reminder BOOLEAN NOT NULL DEFAULT true,
    -- The clock time a personal reminder fires, read in the USER's timezone,
    -- not the couple's (FR-NOTF-002). The couple's zone decides which week it
    -- is; this decides when your own phone buzzes.
    reminder_time   TIME NOT NULL DEFAULT '19:00',
    event_reminders BOOLEAN NOT NULL DEFAULT true,
    important_dates BOOLEAN NOT NULL DEFAULT true,
    appreciation    BOOLEAN NOT NULL DEFAULT true,
    journal         BOOLEAN NOT NULL DEFAULT true,
    -- Off until asked for: a goal or a challenge is something you opt into
    -- following, not something that should start buzzing on its own.
    goals           BOOLEAN NOT NULL DEFAULT false,
    challenges      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per browser that has agreed to receive push. The endpoint is the
-- push service's own identifier for that browser, so it is what uniqueness
-- hangs on: re-subscribing the same browser replaces the keys rather than
-- collecting a second row that would send everything twice.
CREATE TABLE push_subscriptions (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    endpoint     TEXT NOT NULL,
    -- The browser's public key and auth secret, needed to encrypt a payload
    -- that only that browser can read (RFC 8291). Useless to anyone else.
    p256dh       TEXT NOT NULL,
    auth         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Set when a send succeeds; a send that comes back 404 or 410 deletes the
    -- row instead (FR-NOTF-004).
    last_sent_at TIMESTAMPTZ NULL,
    CONSTRAINT push_subscriptions_endpoint_key UNIQUE (endpoint)
);

CREATE INDEX push_subscriptions_user_idx ON push_subscriptions (user_id);

-- +goose Down
DROP TABLE push_subscriptions;
DROP TABLE notification_preferences;
