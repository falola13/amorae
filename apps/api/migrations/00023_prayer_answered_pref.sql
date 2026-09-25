-- +goose Up
-- A switch for hearing that your partner marked a prayer answered.
--
-- Defaults to on, with appreciation and journal rather than with goals and
-- challenges: those two default off because following one is something you
-- opt into, whereas this is your partner telling you something good happened
-- to the two of you. If anything in this app is worth a buzz, it is this.
ALTER TABLE notification_preferences
    ADD COLUMN prayer_answered BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN prayer_answered;
