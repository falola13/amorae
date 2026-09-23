-- +goose Up
-- Leaving a couple ends it for both partners (FR-PAIR-008, Q-09 option (a)):
-- neither one keeps a shared history the other wrote, and neither one can
-- lock the other out of it.
--
-- The row is not deleted at once. Both partners keep reading and exporting
-- for a retention window, and only when that window closes is the couple
-- removed — which takes members, invitations and every couple-owned table
-- with it through ON DELETE CASCADE.
ALTER TABLE couples
    ADD COLUMN dissolved_at TIMESTAMPTZ NULL,
    -- Who ended it. SET NULL rather than CASCADE: if they go on to delete
    -- their account, the other partner's retention window must survive them.
    ADD COLUMN dissolved_by UUID NULL REFERENCES users (id) ON DELETE SET NULL;

-- The sweeper asks one question — "whose window has closed?" — so only
-- dissolved couples belong in the index.
CREATE INDEX couples_dissolved_at_idx ON couples (dissolved_at) WHERE dissolved_at IS NOT NULL;

-- +goose Down
DROP INDEX couples_dissolved_at_idx;
ALTER TABLE couples
    DROP COLUMN dissolved_by,
    DROP COLUMN dissolved_at;
