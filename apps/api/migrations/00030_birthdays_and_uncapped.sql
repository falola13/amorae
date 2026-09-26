-- +goose Up
-- Nobody ever chose six — it just arrived, unpicked, alongside quiet hours
-- (00024_notification_budget.sql). No limit is what "0" already means here,
-- so that becomes the default instead of a number nobody asked for.
ALTER TABLE notification_preferences ALTER COLUMN daily_cap SET DEFAULT 0;
UPDATE notification_preferences SET daily_cap = 0 WHERE daily_cap = 6;

-- A birthday belongs to a profile, not to what a couple decided to keep
-- (BR-DATE-01's mirror image): it's who somebody is, not something either
-- partner wrote down. Year is optional — plenty of people would rather not
-- say, and a month and day are still enough to be told about it every year.
ALTER TABLE users
    ADD COLUMN birth_month SMALLINT NULL CHECK (birth_month BETWEEN 1 AND 12),
    ADD COLUMN birth_day   SMALLINT NULL CHECK (birth_day BETWEEN 1 AND 31),
    ADD COLUMN birth_year  SMALLINT NULL,
    -- Both or neither: a day without a month, or a month without a day,
    -- isn't a birthday.
    ADD CONSTRAINT users_birthday_month_day_together
        CHECK ((birth_month IS NULL) = (birth_day IS NULL)),
    -- A year with no month makes no sense to keep.
    ADD CONSTRAINT users_birth_year_needs_month
        CHECK (birth_year IS NULL OR birth_month IS NOT NULL);

-- +goose Down
ALTER TABLE users
    DROP CONSTRAINT users_birth_year_needs_month,
    DROP CONSTRAINT users_birthday_month_day_together,
    DROP COLUMN birth_year,
    DROP COLUMN birth_day,
    DROP COLUMN birth_month;

-- Restoring the number itself, not the rows the Up migration moved off it —
-- those were never anyone's choice to begin with.
ALTER TABLE notification_preferences ALTER COLUMN daily_cap SET DEFAULT 6;
