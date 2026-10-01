-- +goose Up
-- A couple can have more than one challenge going. The cap on how many is a
-- rule of the service, not of the schema (so it can change without a
-- migration); what the schema keeps is that the same curated challenge is
-- not running twice at once. One the couple wrote themselves is exempt: two
-- of those are two different things that share a filing key.

DROP INDEX challenges_one_active_per_couple;
CREATE UNIQUE INDEX challenges_one_active_per_template
    ON challenges (couple_id, template)
    WHERE status = 'active' AND template <> 'custom';

-- +goose Down
-- Going back means one active challenge per couple again. Where a couple has
-- several going, the one started most recently stays and the others are left
-- as ended — kept with their days, marks and notes, not deleted.
UPDATE challenges ch
SET status = 'ended', ended_at = now(), updated_at = now()
WHERE ch.status = 'active'
  AND EXISTS (
        SELECT 1 FROM challenges newer
        WHERE newer.couple_id = ch.couple_id AND newer.status = 'active'
          AND (newer.created_at, newer.id) > (ch.created_at, ch.id)
      );

DROP INDEX challenges_one_active_per_template;
CREATE UNIQUE INDEX challenges_one_active_per_couple ON challenges (couple_id) WHERE status = 'active';
