-- +goose Up
-- Halfway, and done. Separate from `goals`, which announces every
-- contribution and is off by default for that reason: a crossing happens
-- twice in a goal's life and is worth hearing about.
ALTER TABLE notification_preferences
    ADD COLUMN goal_milestones BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE notification_preferences DROP COLUMN goal_milestones;
