-- +goose Up
-- Challenges are kept, not thrown away. One that is finished together, or
-- left early, stays as part of the couple's story with its days, marks and
-- notes — only one at a time is still going.

ALTER TABLE challenges
    -- 'active' | 'finished' | 'ended'. Everything that existed before this
    -- was still going, so that is what it becomes.
    ADD COLUMN status     TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'finished', 'ended')),
    -- When it stopped being active, by finishing or by being left.
    ADD COLUMN ended_at   TIMESTAMPTZ NULL,
    -- Who started it. Null once that person is gone, not a reason to lose it.
    ADD COLUMN created_by UUID NULL REFERENCES users (id) ON DELETE SET NULL;

-- One going at a time; any number kept. "Current" still has to mean
-- something, but it no longer means the only one there has ever been.
DROP INDEX challenges_one_live_per_couple;
CREATE UNIQUE INDEX challenges_one_active_per_couple ON challenges (couple_id) WHERE status = 'active';

CREATE INDEX challenges_couple_idx ON challenges (couple_id, ended_at);

-- A note on a day is a person's own, next to their mark. Allowed without a
-- mark (a thought on a day they have not settled), so the mark can be empty.
ALTER TABLE challenge_progress ALTER COLUMN mark DROP NOT NULL;
ALTER TABLE challenge_progress ADD COLUMN note TEXT NOT NULL DEFAULT '';

-- What each of them took from it, once it is over. One per person.
CREATE TABLE challenge_reflections (
    challenge_id UUID NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (challenge_id, user_id)
);

-- +goose Down
-- Going back means one challenge per couple again, so the kept ones cannot
-- come with it: this loses that history by nature.
DROP TABLE challenge_reflections;

DELETE FROM challenge_progress WHERE mark IS NULL;
ALTER TABLE challenge_progress DROP COLUMN note;
ALTER TABLE challenge_progress ALTER COLUMN mark SET NOT NULL;

DELETE FROM challenges WHERE status <> 'active';
DROP INDEX challenges_couple_idx;
DROP INDEX challenges_one_active_per_couple;
CREATE UNIQUE INDEX challenges_one_live_per_couple ON challenges (couple_id);

ALTER TABLE challenges
    DROP COLUMN created_by,
    DROP COLUMN ended_at,
    DROP COLUMN status;
