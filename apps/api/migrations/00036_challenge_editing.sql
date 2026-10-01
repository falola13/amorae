-- +goose Up
-- A challenge can be "just me": the other partner can read it but not mark,
-- note, edit, end or reflect on it, and it finishes when its creator has
-- answered every day. 'together' is how every challenge so far behaved.
ALTER TABLE challenges
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'together' CHECK (kind IN ('together', 'mine'));

-- Whether the other partner hears that one was started. Their own switch, on
-- by default, the same as partner_events.
ALTER TABLE notification_preferences
    ADD COLUMN partner_challenges BOOLEAN NOT NULL DEFAULT true;

-- "The same library challenge can't be running twice for one person" cannot
-- be one index any more: a together challenge counts for both of them and a
-- mine one for its creator alone, and an index cannot look across kinds. The
-- service enforces the whole rule inside the couple-locked transaction that
-- starts a challenge; these two are the backstop for the plain cases (two
-- together ones, or two of the same person's own), so a race that somehow got
-- past the lock still cannot write a duplicate. A challenge the couple wrote
-- themselves ('custom') is exempt, as before.
DROP INDEX challenges_one_active_per_template;
CREATE UNIQUE INDEX challenges_one_active_together_per_template
    ON challenges (couple_id, template)
    WHERE status = 'active' AND template <> 'custom' AND kind = 'together';
CREATE UNIQUE INDEX challenges_one_active_mine_per_template
    ON challenges (couple_id, template, created_by)
    WHERE status = 'active' AND template <> 'custom' AND kind = 'mine';

-- +goose Down
-- Going back to one index means no library challenge twice per couple. Where
-- a couple has the same one active more than once (a "mine" for each of them,
-- or a "mine" beside a together), the one started most recently stays and the
-- others are left as ended — kept with their days, marks and notes, not
-- deleted — before the old index can be put back.
UPDATE challenges ch
SET status = 'ended', ended_at = now(), updated_at = now()
WHERE ch.status = 'active' AND ch.template <> 'custom'
  AND EXISTS (
        SELECT 1 FROM challenges newer
        WHERE newer.couple_id = ch.couple_id AND newer.template = ch.template
          AND newer.status = 'active'
          AND (newer.created_at, newer.id) > (ch.created_at, ch.id)
      );

DROP INDEX challenges_one_active_mine_per_template;
DROP INDEX challenges_one_active_together_per_template;
CREATE UNIQUE INDEX challenges_one_active_per_template
    ON challenges (couple_id, template)
    WHERE status = 'active' AND template <> 'custom';

ALTER TABLE notification_preferences DROP COLUMN IF EXISTS partner_challenges;
ALTER TABLE challenges DROP COLUMN kind;
