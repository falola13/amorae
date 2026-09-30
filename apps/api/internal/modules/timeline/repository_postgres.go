package timeline

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Timeline is the couple's shared story, assembled from eight tables in one
// query rather than one round trip per source. Every branch below joins the
// couples row it belongs to for that couple's own timezone (DEC-27) — the
// same "join couples for c.timezone" the journal and prayers repositories
// already do — since a couple's timezone can only be read from that row.
//
// `types` scopes the union to one Filter's sources (see TypesFor); `before`,
// when not nil, pages strictly earlier than that instant; `limit` is exactly
// how many rows to return, chosen by Service (which asks for one extra of
// its own accord).
func (r *PostgresRepository) Timeline(
	ctx context.Context, coupleID uuid.UUID, types []Type, before *time.Time, limit int,
) ([]Item, error) {
	typeNames := make([]string, len(types))
	for i, t := range types {
		typeNames[i] = string(t)
	}

	rows, err := r.db.Q(ctx).Query(ctx, timelineQuery, coupleID, typeNames, before, limit)
	if err != nil {
		return nil, fmt.Errorf("loading timeline: %w", err)
	}
	defer rows.Close()

	out := []Item{}
	for rows.Next() {
		var it Item
		var typ string
		var photoID string
		var actorID *uuid.UUID
		var version *time.Time
		if err := rows.Scan(&typ, &it.ID, &it.At, &it.Date, &it.Title, &it.Sub, &it.Path,
			&photoID, &actorID, &version); err != nil {
			return nil, fmt.Errorf("scanning timeline item: %w", err)
		}
		it.Type = Type(typ)
		it.PhotoID = photoID
		if actorID != nil {
			it.ActorID = *actorID
		}
		if version != nil {
			it.Version = *version
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading timeline: %w", err)
	}
	return out, nil
}

// timelineQuery unions every source the timeline draws from into one common
// shape (type, id, at, date, title, sub, path, photo_id, actor_id, version),
// then filters and pages the result. Each branch is commented with what "at"
// and "sub" mean for that source; the outer SELECT is the only place paging
// and filtering happen.
const timelineQuery = `
WITH items AS (
	-- Every PAST published prayer week — "at" is the moment the week ends,
	-- so a week never appears until it's over, and never as the current one.
	SELECT 'prayer_week'::text AS type, w.id,
	       ((w.week_start + 7)::timestamp AT TIME ZONE c.timezone) AS at,
	       w.week_start + 6 AS date,
	       'Prayer week'::text AS title,
	       ((SELECT count(*) FROM prayer_points pp WHERE pp.week_id = w.id)::text || ' prayers') AS sub,
	       ('/history/' || w.id::text) AS path,
	       NULL::text AS photo_id, NULL::uuid AS actor_id, NULL::timestamptz AS version
	FROM prayer_weeks w
	JOIN couples c ON c.id = w.couple_id
	WHERE w.couple_id = $1
	  AND w.status = 'published'
	  AND w.week_start < ((now() AT TIME ZONE c.timezone)::date
	                       - EXTRACT(DOW FROM (now() AT TIME ZONE c.timezone)::date)::int)

	UNION ALL

	-- A prayer marked answered, from whichever week it belongs to.
	SELECT 'prayer_answered', p.id,
	       p.answered_at,
	       (p.answered_at AT TIME ZONE c.timezone)::date,
	       p.title,
	       CASE WHEN p.answer_note <> '' THEN 'Answered · ' || left(p.answer_note, 80) ELSE 'Answered' END,
	       '/prayers/answered',
	       NULL::text, p.answered_by, NULL::timestamptz
	FROM prayer_points p
	JOIN prayer_weeks w ON w.id = p.week_id
	JOIN couples c ON c.id = w.couple_id
	WHERE w.couple_id = $1 AND p.answered_at IS NOT NULL

	UNION ALL

	-- A kept moment. "at" is midnight of the day it happened, nudged by the
	-- time of day it was actually written down (created_at's own clock
	-- time), so two memories logged for the same date still separate in a
	-- stable, meaningful order rather than tying on the same instant.
	SELECT 'memory', m.id,
	       (m.date::timestamp AT TIME ZONE c.timezone) + (m.created_at - date_trunc('day', m.created_at)),
	       m.date,
	       m.title,
	       COALESCE(NULLIF(m.location, ''), NULLIF(left(COALESCE(m.note, ''), 80), '')),
	       '/together/memories',
	       NULLIF(m.photo_id, ''), NULL::uuid, m.updated_at
	FROM memories m
	JOIN couples c ON c.id = m.couple_id
	WHERE m.couple_id = $1

	UNION ALL

	-- Anything already done, or whose day has passed — an event still ahead
	-- of the couple belongs on their calendar, not their history.
	SELECT 'event', e.id,
	       ((e.date::timestamp + COALESCE(e.start_time, '00:00'::time)) AT TIME ZONE c.timezone),
	       e.date,
	       e.title,
	       CASE
	           WHEN e.kind = 'mine' AND u.display_name IS NOT NULL THEN 'Just ' || u.display_name
	           WHEN NULLIF(e.location, '') IS NOT NULL THEN e.location
	           ELSE 'Done'
	       END,
	       ('/together/events/' || e.id::text),
	       NULL::text, e.created_by, NULL::timestamptz
	FROM events e
	JOIN couples c ON c.id = e.couple_id
	LEFT JOIN users u ON u.id = e.created_by
	WHERE e.couple_id = $1
	  AND (e.done OR e.date < (now() AT TIME ZONE c.timezone)::date)

	UNION ALL

	-- A goal the couple finished. goals has no completed-at column — the
	-- moment it was marked done is the last time it was touched, updated_at.
	SELECT 'goal', g.id,
	       g.updated_at,
	       (g.updated_at AT TIME ZONE c.timezone)::date,
	       g.title, 'Reached'::text, '/together/goals',
	       NULL::text, NULL::uuid, NULL::timestamptz
	FROM goals g
	JOIN couples c ON c.id = g.couple_id
	WHERE g.couple_id = $1 AND g.done

	UNION ALL

	SELECT 'journal', j.id,
	       j.created_at,
	       (j.created_at AT TIME ZONE c.timezone)::date,
	       j.tag::text, left(j.text, 80), '/together/journal',
	       NULL::text, j.author_id, NULL::timestamptz
	FROM journal_entries j
	JOIN couples c ON c.id = j.couple_id
	WHERE j.couple_id = $1

	UNION ALL

	SELECT 'appreciation', a.id,
	       a.created_at,
	       (a.created_at AT TIME ZONE c.timezone)::date,
	       'Appreciation'::text, left(a.text, 80), '/together/appreciation',
	       NULL::text, a.from_id, NULL::timestamptz
	FROM appreciations a
	JOIN couples c ON c.id = a.couple_id
	WHERE a.couple_id = $1

	UNION ALL

	-- A challenge that is over, finished together or left early. 'at' is the
	-- moment it stopped being active.
	SELECT 'challenge', ch.id,
	       ch.ended_at,
	       (ch.ended_at AT TIME ZONE c.timezone)::date,
	       ch.title,
	       CASE WHEN ch.status = 'finished' THEN 'Finished together' ELSE 'Ended early' END,
	       ('/together/challenges/' || ch.id::text),
	       NULL::text, NULL::uuid, NULL::timestamptz
	FROM challenges ch
	JOIN couples c ON c.id = ch.couple_id
	WHERE ch.couple_id = $1 AND ch.status IN ('finished', 'ended') AND ch.ended_at IS NOT NULL
)
-- Stable ordering: "at" alone can tie (an event with no start time, two
-- rows sharing an instant), so type and id break the tie the same way on
-- every page — what makes offset-free cursor paging safe.
SELECT type, id, at, date, title, sub, path, COALESCE(photo_id, ''), actor_id, version
FROM items
WHERE type = ANY($2)
  AND ($3::timestamptz IS NULL OR at < $3::timestamptz)
ORDER BY at DESC, type, id
LIMIT $4
`
