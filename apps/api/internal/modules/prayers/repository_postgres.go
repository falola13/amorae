package prayers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Every method here takes a couple id, never a user id — "who is calling"
// to "which couple" is the couples module's join.

// EnsureWeek returns the id of the couple's week beginning weekStart,
// creating it with `setter` as its setter if it is not there yet.
// UNIQUE (couple_id, week_start) makes concurrent calls race-safe — the
// insert that loses is a no-op, not an error.
func (r *PostgresRepository) EnsureWeek(
	ctx context.Context, coupleID uuid.UUID, weekStart time.Time, setter uuid.UUID, at time.Time,
) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating week id: %w", err)
	}

	var weekID uuid.UUID
	err = r.db.Q(ctx).QueryRow(ctx, `
		INSERT INTO prayer_weeks (id, couple_id, week_start, setter_user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'draft', $5, $5)
		ON CONFLICT (couple_id, week_start) DO NOTHING
		RETURNING id
	`, id, coupleID, weekStart, setter, at).Scan(&weekID)
	if err == nil {
		return weekID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, fmt.Errorf("creating prayer week: %w", err)
	}

	// DO NOTHING returns no row: "no rows" means somebody else won the race.
	if err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id FROM prayer_weeks WHERE couple_id = $1 AND week_start = $2
	`, coupleID, weekStart).Scan(&weekID); err != nil {
		return uuid.UUID{}, fmt.Errorf("reading prayer week: %w", err)
	}
	return weekID, nil
}

// Record is a stored week plus the parts of its state that belong to one
// person rather than the couple: completions and reflections.
type Record struct {
	Week
	// user id -> the points that user has prayed on any day this week — the
	// "ever this week" view History and the lock rule both want.
	Completed map[uuid.UUID][]uuid.UUID
	// day (as WeekStart's date-only representation) -> user id -> the points
	// that user prayed that specific day — what "today"'s my/partner lists,
	// and the days breakdown, are built from.
	ByDay map[time.Time]map[uuid.UUID][]uuid.UUID
	// user id -> what that person wrote about the week.
	Reflections map[uuid.UUID]string
}

// PrayedByOthers is the set of point ids somebody other than `except` has
// prayed, on any day this week — the input CanEditPoints needs.
func (rec Record) PrayedByOthers(except uuid.UUID) map[uuid.UUID]bool {
	var out map[uuid.UUID]bool
	for userID, points := range rec.Completed {
		if userID == except {
			continue
		}
		for _, pointID := range points {
			if out == nil {
				out = make(map[uuid.UUID]bool, len(points))
			}
			out[pointID] = true
		}
	}
	return out
}

func (r *PostgresRepository) WeekByID(ctx context.Context, coupleID, weekID uuid.UUID) (Record, error) {
	return r.one(ctx, `WHERE w.couple_id = $1 AND w.id = $2`, coupleID, weekID)
}

func (r *PostgresRepository) WeekStarting(ctx context.Context, coupleID uuid.UUID, weekStart time.Time) (Record, error) {
	return r.one(ctx, `WHERE w.couple_id = $1 AND w.week_start = $2`, coupleID, weekStart)
}

// History is the couple's earlier weeks, newest first.
func (r *PostgresRepository) History(ctx context.Context, coupleID uuid.UUID, before time.Time) ([]Record, error) {
	return r.load(ctx, `WHERE w.couple_id = $1 AND w.week_start < $2`, coupleID, before)
}

func (r *PostgresRepository) one(ctx context.Context, where string, args ...any) (Record, error) {
	records, err := r.load(ctx, where, args...)
	if err != nil {
		return Record{}, err
	}
	if len(records) == 0 {
		return Record{}, ErrNotFound
	}
	return records[0], nil
}

// load reads whole weeks — points, completions and reflections — in a fixed
// four queries regardless of result size, rather than N+1 per week.
func (r *PostgresRepository) load(ctx context.Context, where string, args ...any) ([]Record, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT w.id, w.couple_id, w.week_start, w.setter_user_id, w.status, w.published_at, w.published_by
		FROM prayer_weeks w `+where+`
		ORDER BY w.week_start DESC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("loading prayer weeks: %w", err)
	}

	var records []Record
	at := map[uuid.UUID]int{} // week id -> where it sits in records
	for rows.Next() {
		var rec Record
		var publishedBy *uuid.UUID
		if err := rows.Scan(&rec.ID, &rec.CoupleID, &rec.WeekStart, &rec.SetterUserID,
			&rec.Status, &rec.PublishedAt, &publishedBy); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning prayer week: %w", err)
		}
		rec.PublishedBy = publishedBy
		rec.Completed = map[uuid.UUID][]uuid.UUID{}
		rec.ByDay = map[time.Time]map[uuid.UUID][]uuid.UUID{}
		rec.Reflections = map[uuid.UUID]string{}
		at[rec.ID] = len(records)
		records = append(records, rec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading prayer weeks: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	weekIDs := make([]uuid.UUID, 0, len(records))
	for _, rec := range records {
		weekIDs = append(weekIDs, rec.ID)
	}

	// Points, ordered so the setter's arrangement survives the round trip.
	pointRows, err := r.db.Q(ctx).Query(ctx, `
		SELECT p.id, p.week_id, p.position, p.title, p.body, p.weekdays,
		       COALESCE(p.scripture, ''), COALESCE(p.verse, ''),
		       p.answered_at, (p.answered_at AT TIME ZONE c.timezone)::date,
		       p.answered_by, p.answer_note
		FROM prayer_points p
		JOIN prayer_weeks w ON w.id = p.week_id
		JOIN couples c ON c.id = w.couple_id
		WHERE p.week_id = ANY($1)
		ORDER BY p.week_id, p.position
	`, weekIDs)
	if err != nil {
		return nil, fmt.Errorf("loading prayer points: %w", err)
	}
	weekOfPoint := map[uuid.UUID]uuid.UUID{} // point id -> week id, for completions below
	for pointRows.Next() {
		var p Point
		var weekID uuid.UUID
		var answeredBy *uuid.UUID
		if err := pointRows.Scan(&p.ID, &weekID, &p.Position, &p.Title, &p.Body, &p.Weekdays,
			&p.Scripture, &p.Verse, &p.AnsweredAt, &p.AnsweredOn, &answeredBy,
			&p.AnswerNote); err != nil {
			pointRows.Close()
			return nil, fmt.Errorf("scanning prayer point: %w", err)
		}
		if answeredBy != nil {
			p.AnsweredBy = *answeredBy
		}
		weekOfPoint[p.ID] = weekID
		i := at[weekID]
		records[i].Points = append(records[i].Points, p)
	}
	pointRows.Close()
	if err := pointRows.Err(); err != nil {
		return nil, fmt.Errorf("loading prayer points: %w", err)
	}

	completionRows, err := r.db.Q(ctx).Query(ctx, `
		SELECT c.point_id, c.user_id, c.prayed_on
		FROM prayer_completions c
		JOIN prayer_points p ON p.id = c.point_id
		WHERE p.week_id = ANY($1)
	`, weekIDs)
	if err != nil {
		return nil, fmt.Errorf("loading completions: %w", err)
	}
	// Completed collapses every day into one "prayed this week" entry per
	// point, so the same point prayed on two different days must not appear
	// in it twice.
	seen := map[uuid.UUID]map[uuid.UUID]bool{} // point id -> user id -> already counted
	for completionRows.Next() {
		var pointID, userID uuid.UUID
		var prayedOn time.Time
		if err := completionRows.Scan(&pointID, &userID, &prayedOn); err != nil {
			completionRows.Close()
			return nil, fmt.Errorf("scanning completion: %w", err)
		}
		i := at[weekOfPoint[pointID]]

		if seen[pointID] == nil {
			seen[pointID] = map[uuid.UUID]bool{}
		}
		if !seen[pointID][userID] {
			seen[pointID][userID] = true
			records[i].Completed[userID] = append(records[i].Completed[userID], pointID)
		}

		byDay := records[i].ByDay
		if byDay[prayedOn] == nil {
			byDay[prayedOn] = map[uuid.UUID][]uuid.UUID{}
		}
		byDay[prayedOn][userID] = append(byDay[prayedOn][userID], pointID)
	}
	completionRows.Close()
	if err := completionRows.Err(); err != nil {
		return nil, fmt.Errorf("loading completions: %w", err)
	}

	reflectionRows, err := r.db.Q(ctx).Query(ctx, `
		SELECT week_id, user_id, body FROM prayer_reflections WHERE week_id = ANY($1)
	`, weekIDs)
	if err != nil {
		return nil, fmt.Errorf("loading reflections: %w", err)
	}
	for reflectionRows.Next() {
		var weekID, userID uuid.UUID
		var body string
		if err := reflectionRows.Scan(&weekID, &userID, &body); err != nil {
			reflectionRows.Close()
			return nil, fmt.Errorf("scanning reflection: %w", err)
		}
		records[at[weekID]].Reflections[userID] = body
	}
	reflectionRows.Close()
	if err := reflectionRows.Err(); err != nil {
		return nil, fmt.Errorf("loading reflections: %w", err)
	}

	return records, nil
}

// FirstWeekStart anchors the setter rotation; ok is false before the couple
// has had any week.
func (r *PostgresRepository) FirstWeekStart(ctx context.Context, coupleID uuid.UUID) (time.Time, bool, error) {
	var first *time.Time
	if err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT min(week_start) FROM prayer_weeks WHERE couple_id = $1
	`, coupleID).Scan(&first); err != nil {
		return time.Time{}, false, fmt.Errorf("finding first prayer week: %w", err)
	}
	if first == nil {
		return time.Time{}, false, nil
	}
	return *first, true, nil
}

// WeekOfPoint is the week a point belongs to, and doubles as the permission
// check: a point of somebody else's couple is simply not found.
func (r *PostgresRepository) WeekOfPoint(ctx context.Context, coupleID, pointID uuid.UUID) (uuid.UUID, error) {
	var weekID uuid.UUID
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT p.week_id FROM prayer_points p
		JOIN prayer_weeks w ON w.id = p.week_id
		WHERE p.id = $1 AND w.couple_id = $2
	`, pointID, coupleID).Scan(&weekID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, ErrNotFound
	}
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("finding the point's week: %w", err)
	}
	return weekID, nil
}

// ReplacePoints makes the week's points exactly `points`, in the order
// given. Existing points are matched by id and kept (so completions
// survive a reorder); a dropped point is deleted, cascading its
// completions. An id the week doesn't already own is ignored, not honoured
// — primary keys are the server's to choose.
func (r *PostgresRepository) ReplacePoints(ctx context.Context, weekID uuid.UUID, points []Point, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		existing, err := r.pointIDsOf(ctx, weekID)
		if err != nil {
			return err
		}

		// Move every position out of range first: UNIQUE (week_id, position)
		// is checked statement by statement, so swapping two would collide.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE prayer_points SET position = -position - 1 WHERE week_id = $1
		`, weekID); err != nil {
			return fmt.Errorf("clearing positions: %w", err)
		}

		keep := make([]uuid.UUID, 0, len(points))
		for _, p := range points {
			if _, known := existing[p.ID]; known {
				if _, err := r.db.Q(ctx).Exec(ctx, `
					UPDATE prayer_points
					SET position = $3, title = $4, body = $5, scripture = $6, verse = $7, weekdays = $8, updated_at = $9
					WHERE id = $1 AND week_id = $2
				`, p.ID, weekID, p.Position, p.Title, p.Body,
					nullIfEmpty(p.Scripture), nullIfEmpty(p.Verse), p.Weekdays, at); err != nil {
					return fmt.Errorf("updating prayer point: %w", err)
				}
				keep = append(keep, p.ID)
				continue
			}

			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generating point id: %w", err)
			}
			if _, err := r.db.Q(ctx).Exec(ctx, `
				INSERT INTO prayer_points (id, week_id, position, title, body, scripture, verse, weekdays, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
			`, id, weekID, p.Position, p.Title, p.Body,
				nullIfEmpty(p.Scripture), nullIfEmpty(p.Verse), p.Weekdays, at); err != nil {
				return fmt.Errorf("adding prayer point: %w", err)
			}
			keep = append(keep, id)
		}

		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM prayer_points WHERE week_id = $1 AND NOT (id = ANY($2))
		`, weekID, keep); err != nil {
			return fmt.Errorf("removing dropped prayer points: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) pointIDsOf(ctx context.Context, weekID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT id FROM prayer_points WHERE week_id = $1`, weekID)
	if err != nil {
		return nil, fmt.Errorf("listing prayer points: %w", err)
	}
	defer rows.Close()

	ids := map[uuid.UUID]struct{}{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning prayer point id: %w", err)
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing prayer points: %w", err)
	}
	return ids, nil
}

// Publish shares the week, recording who did it. Publishing an
// already-published week is a no-op, so a republish by the other partner
// never steals credit for having shared it first.
func (r *PostgresRepository) Publish(ctx context.Context, weekID, publisherID uuid.UUID, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE prayer_weeks
		SET status = 'published', published_at = $2, published_by = $3, updated_at = $2
		WHERE id = $1 AND status = 'draft'
	`, weekID, at, publisherID); err != nil {
		return fmt.Errorf("publishing prayer week: %w", err)
	}
	return nil
}

// SetCompletion marks or unmarks one point for one person, for one specific
// day. Marking the same day twice is a no-op, which is what makes the
// offline queue safe to replay.
func (r *PostgresRepository) SetCompletion(ctx context.Context, pointID, userID uuid.UUID, prayedOn time.Time, done bool, at time.Time) error {
	if !done {
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM prayer_completions WHERE point_id = $1 AND user_id = $2 AND prayed_on = $3
		`, pointID, userID, prayedOn); err != nil {
			return fmt.Errorf("clearing completion: %w", err)
		}
		return nil
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO prayer_completions (point_id, user_id, prayed_on, completed_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (point_id, user_id, prayed_on) DO NOTHING
	`, pointID, userID, prayedOn, at); err != nil {
		return fmt.Errorf("recording completion: %w", err)
	}
	return nil
}

// SetAnswered records — or takes back — the fact that a prayer was
// answered. Unanswering clears the note too, since it no longer applies.
func (r *PostgresRepository) SetAnswered(
	ctx context.Context, pointID, userID uuid.UUID, answered bool, note string, at time.Time,
) error {
	if !answered {
		_, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE prayer_points
			SET answered_at = NULL, answered_by = NULL, answer_note = '', updated_at = $2
			WHERE id = $1
		`, pointID, at)
		if err != nil {
			return fmt.Errorf("clearing answered prayer: %w", err)
		}
		return nil
	}
	// COALESCE keeps the original answered_at on a re-save, so editing the
	// note doesn't re-date the answer.
	_, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE prayer_points
		SET answered_at = COALESCE(answered_at, $3), answered_by = $2,
		    answer_note = $4, updated_at = $3
		WHERE id = $1
	`, pointID, userID, at, note)
	if err != nil {
		return fmt.Errorf("marking prayer answered: %w", err)
	}
	return nil
}

// Answered is every answered prayer a couple has, newest first.
func (r *PostgresRepository) Answered(ctx context.Context, coupleID uuid.UUID) ([]Answered, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT p.id, p.position, p.title, p.body, COALESCE(p.scripture, ''), COALESCE(p.verse, ''),
		       p.answered_at, (p.answered_at AT TIME ZONE c.timezone)::date,
		       p.answered_by, p.answer_note, w.id, w.week_start
		FROM prayer_points p
		JOIN prayer_weeks w ON w.id = p.week_id
		JOIN couples c ON c.id = w.couple_id
		WHERE w.couple_id = $1 AND p.answered_at IS NOT NULL
		ORDER BY p.answered_at DESC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading answered prayers: %w", err)
	}
	defer rows.Close()

	out := []Answered{}
	for rows.Next() {
		var a Answered
		var answeredBy *uuid.UUID
		if err := rows.Scan(&a.ID, &a.Position, &a.Title, &a.Body, &a.Scripture, &a.Verse,
			&a.AnsweredAt, &a.AnsweredOn, &answeredBy, &a.AnswerNote,
			&a.WeekID, &a.WeekStart); err != nil {
			return nil, fmt.Errorf("scanning answered prayer: %w", err)
		}
		if answeredBy != nil {
			a.AnsweredBy = *answeredBy
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading answered prayers: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) SetReflection(ctx context.Context, weekID, userID uuid.UUID, body string, at time.Time) error {
	if body == "" {
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM prayer_reflections WHERE week_id = $1 AND user_id = $2
		`, weekID, userID); err != nil {
			return fmt.Errorf("clearing reflection: %w", err)
		}
		return nil
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO prayer_reflections (week_id, user_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (week_id, user_id) DO UPDATE SET body = EXCLUDED.body, updated_at = EXCLUDED.updated_at
	`, weekID, userID, body, at); err != nil {
		return fmt.Errorf("saving reflection: %w", err)
	}
	return nil
}

// nullIfEmpty converts domain empty string to SQL NULL for nullable columns.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
