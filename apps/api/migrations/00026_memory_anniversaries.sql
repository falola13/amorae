-- +goose Up
-- Moments kept a year ago, offered back on the day.
ALTER TABLE notification_preferences
    ADD COLUMN memories BOOLEAN NOT NULL DEFAULT TRUE;

-- The anniversary query asks for one month of a couple's memories at a time.
CREATE INDEX memories_couple_month_idx
    ON memories (couple_id, (EXTRACT(MONTH FROM date)));

-- +goose Down
DROP INDEX memories_couple_month_idx;
ALTER TABLE notification_preferences DROP COLUMN memories;
