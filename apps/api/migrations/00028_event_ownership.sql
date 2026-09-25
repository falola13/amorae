-- +goose Up
-- Who made a thing, and who it is for. An event is either "together" — both
-- partners may edit it, and both hear about it — or "mine" — only its
-- creator may touch it, and only its creator is told about it. Existing
-- events predate the idea: created_by stays NULL and kind stays 'together',
-- which is exactly how they already behaved.
ALTER TABLE events
    ADD COLUMN created_by UUID NULL REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE events
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'together' CHECK (kind IN ('together', 'mine'));

ALTER TABLE notification_preferences
    ADD COLUMN event_followups BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE notification_preferences
    ADD COLUMN partner_events BOOLEAN NOT NULL DEFAULT true;
-- '1 hour before' matches what the create-event screen has always offered
-- as the starting choice; '' is still a valid pick — it just has to be made,
-- not defaulted into.
ALTER TABLE notification_preferences
    ADD COLUMN default_event_reminder TEXT NOT NULL DEFAULT '1 hour before';

-- event_followups starts wherever event_reminders already was: today
-- KindEventOver is gated by event_reminders, so somebody who turned
-- reminders off should not suddenly start being asked "how was it?" again.
UPDATE notification_preferences SET event_followups = event_reminders;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN default_event_reminder;
ALTER TABLE notification_preferences DROP COLUMN partner_events;
ALTER TABLE notification_preferences DROP COLUMN event_followups;
ALTER TABLE events DROP COLUMN kind;
ALTER TABLE events DROP COLUMN created_by;
