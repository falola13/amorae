-- +goose Up
-- A ceiling on how much the app may interrupt one person, and the shared
-- moments worth interrupting for.

ALTER TABLE notification_preferences
    -- Both null means no quiet hours. A window that wraps midnight is normal
    -- (22:00 to 07:00), so "quiet" is not simply from < to.
    ADD COLUMN quiet_from TIME,
    ADD COLUMN quiet_to   TIME,
    -- Notifications per person per local day; 0 means no limit.
    ADD COLUMN daily_cap  SMALLINT NOT NULL DEFAULT 6 CHECK (daily_cap >= 0),
    -- "You both did it" — the only notification that is about the two of
    -- them rather than one of them.
    ADD COLUMN together   BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE notification_preferences SET quiet_from = '22:00', quiet_to = '07:00';

-- Counting a person's day needs their rows by time, not their whole history.
CREATE INDEX notification_sends_user_sent_idx ON notification_sends (user_id, sent_at);

-- +goose Down
DROP INDEX notification_sends_user_sent_idx;
ALTER TABLE notification_preferences
    DROP COLUMN quiet_from,
    DROP COLUMN quiet_to,
    DROP COLUMN daily_cap,
    DROP COLUMN together;
