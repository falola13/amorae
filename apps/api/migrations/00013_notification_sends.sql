-- +goose Up
-- What has already been sent, so nothing is sent twice.
--
-- The worker ticks hourly, can be restarted mid-tick, and may one day run in
-- more than one place. Rather than have it remember what it did, the record
-- of each send is a row whose primary key is what makes that send unique —
-- so a second attempt loses at the insert, exactly as a repeated prayer
-- completion does (DEC-28). Nothing has to be remembered between ticks.
CREATE TABLE notification_sends (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Which notification: 'new_week', 'prayer_reminder'.
    kind    TEXT NOT NULL,
    -- What makes this one distinct within its kind: the week's id for
    -- 'new_week', the person's own local date for 'prayer_reminder'. A date
    -- rather than a timestamp because "today's reminder" is a calendar fact,
    -- and it is read in that person's own timezone.
    key     TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind, key)
);

-- Old rows are of no use once their moment has passed; this is what a
-- periodic tidy would read.
CREATE INDEX notification_sends_sent_at_idx ON notification_sends (sent_at);

-- +goose Down
DROP TABLE notification_sends;
