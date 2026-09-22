-- +goose Up
-- Installing the PWA and allowing notifications happen on one person's phone,
-- so these live on the membership row rather than on couples.
--
-- There is deliberately no column for the couple step: being a member of a
-- couple IS that step's completed state, and a separate flag could only ever
-- drift out of agreement with this row's existence.
ALTER TABLE couple_members
ADD COLUMN onboarding_install BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN onboarding_notifications BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE couple_members
DROP COLUMN onboarding_install,
DROP COLUMN onboarding_notifications;
