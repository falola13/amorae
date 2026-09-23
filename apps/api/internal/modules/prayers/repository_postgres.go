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

// Every method here takes a couple id, never a user id. Turning "who is
// calling" into "which couple" is the couples module's join, and two modules
// owning the same join is two modules that will disagree about it one day.

// EnsureWeek returns the id of the couple's week beginning weekStart,
// creating it with `setter` as its setter if it is not there yet.
//
// Named for the guarantee rather than the action, because the guarantee is
// the point: this runs on every read of the current week, and two readers in
// the same instant must end up with one week between them, not two.
// UNIQUE (couple_id, week_start) is what enforces that — the insert that
// loses is a no-op, not an error.
//
// The setter is passed in rather than worked out here: whose turn it is is a
// rule (SetterFor), and rules do not live in a repository.
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

	// DO NOTHING returns no row, so "no rows" here means somebody else got
	// there first — a success, not a failure. Read theirs.
	if err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id FROM prayer_weeks WHERE couple_id = $1 AND week_start = $2
	`, coupleID, weekStart).Scan(&weekID); err != nil {
		return uuid.UUID{}, fmt.Errorf("reading prayer week: %w", err)
	}
	return weekID, nil
}

// Record is a stored week plus the parts of its state that belong to one
// person rather than to the couple: who has completed which points, and what
// each of them wrote about the week.
//
// Week itself stays exactly as the rules see it. CanEditPoints takes
// partnerHasCompleted as an argument instead of reading it off the week, and
// that separation is worth keeping — it is why the rules can be tested
// without any of this.
type Record struct {
	Week
	// user id -> the points that user has completed.
	Completed map[uuid.UUID][]uuid.UUID
	// user id -> what that person wrote about the week.
	Reflections map[uuid.UUID]string
}

// CompletedByAnyoneBut reports whether someone other than `except` has
// completed any point of this week — the question CanEditPoints asks, in the
// words it asks it.
func (rec Record) CompletedByAnyoneBut(except uuid.UUID) bool {
	for userID, points := range rec.Completed {
		if userID != except && len(points) > 0 {
			return true
		}
	}
	return false
}

// WeekByID loads one week of this couple's, in full.
func (r *PostgresRepository) WeekByID(ctx context.Context, coupleID, weekID uuid.UUID) (Record, error) {
	return r.one(ctx, `WHERE w.couple_id = $1 AND w.id = $2`, coupleID, weekID)
}

// WeekStarting loads the couple's week beginning on weekStart.
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

// load reads whole weeks — points, completions and reflections included — in
// a fixed four queries, whether it returns one week or a year of them.
//
// The obvious version asks for the weeks, then loops asking for each week's
// points, then loops again for its completions. That is fine for one week and
// a hundred round trips for a history screen. Instead each child table is
// fetched once for every week in the result (`= ANY($1)`) and the pieces are
// stitched together here, in memory, where it costs nothing.
func (r *PostgresRepository) load(ctx context.Context, where string, args ...any) ([]Record, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT w.id, w.couple_id, w.week_start, w.setter_user_id, w.status, w.published_at
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
		if err := rows.Scan(&rec.ID, &rec.CoupleID, &rec.WeekStart, &rec.SetterUserID,
			&rec.Status, &rec.PublishedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning prayer week: %w", err)
		}
		rec.Completed = map[uuid.UUID][]uuid.UUID{}
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
		SELECT id, week_id, position, title, body, COALESCE(scripture, ''), COALESCE(verse, '')
		FROM prayer_points WHERE week_id = ANY($1)
		ORDER BY week_id, position
	`, weekIDs)
	if err != nil {
		return nil, fmt.Errorf("loading prayer points: %w", err)
	}
	weekOfPoint := map[uuid.UUID]uuid.UUID{} // point id -> week id, for completions below
	for pointRows.Next() {
		var p Point
		var weekID uuid.UUID
		if err := pointRows.Scan(&p.ID, &weekID, &p.Position, &p.Title, &p.Body,
			&p.Scripture, &p.Verse); err != nil {
			pointRows.Close()
			return nil, fmt.Errorf("scanning prayer point: %w", err)
		}
		weekOfPoint[p.ID] = weekID
		i := at[weekID]
		records[i].Points = append(records[i].Points, p)
	}
	pointRows.Close()
	if err := pointRows.Err(); err != nil {
		return nil, fmt.Errorf("loading prayer points: %w", err)
	}

	// Completions, reached through the points we just read.
	completionRows, err := r.db.Q(ctx).Query(ctx, `
		SELECT c.point_id, c.user_id
		FROM prayer_completions c
		JOIN prayer_points p ON p.id = c.point_id
		WHERE p.week_id = ANY($1)
	`, weekIDs)
	if err != nil {
		return nil, fmt.Errorf("loading completions: %w", err)
	}
	for completionRows.Next() {
		var pointID, userID uuid.UUID
		if err := completionRows.Scan(&pointID, &userID); err != nil {
			completionRows.Close()
			return nil, fmt.Errorf("scanning completion: %w", err)
		}
		i := at[weekOfPoint[pointID]]
		records[i].Completed[userID] = append(records[i].Completed[userID], pointID)
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

// FirstWeekStart is the couple's earliest prayer week, which anchors the
// setter rotation. ok is false before they have had any week at all.
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

// ReplacePoints makes the week's points exactly `points`, in the order given.
//
// Existing points are matched by id and kept, so a reorder does not throw
// away the completions hanging off them. A point the setter has dropped is
// deleted, and its completions cascade with it — which is right: the thing
// people were praying for is gone.
//
// An id the client sends that this week does not already have is ignored
// rather than honoured. Primary keys are the server's to choose.
func (r *PostgresRepository) ReplacePoints(ctx context.Context, weekID uuid.UUID, points []Point, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		existing, err := r.pointIDsOf(ctx, weekID)
		if err != nil {
			return err
		}

		// Move every current position out of range first. UNIQUE
		// (week_id, position) is checked statement by statement, so swapping
		// two points would collide half way through without this.
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
					SET position = $3, title = $4, body = $5, scripture = $6, verse = $7, updated_at = $8
					WHERE id = $1 AND week_id = $2
				`, p.ID, weekID, p.Position, p.Title, p.Body,
					nullIfEmpty(p.Scripture), nullIfEmpty(p.Verse), at); err != nil {
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
				INSERT INTO prayer_points (id, week_id, position, title, body, scripture, verse, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
			`, id, weekID, p.Position, p.Title, p.Body,
				nullIfEmpty(p.Scripture), nullIfEmpty(p.Verse), at); err != nil {
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

// Publish shares the week. Publishing one that is already published is a
// no-op rather than an error: the client may retry, and the second attempt
// should find the world as it wanted it.
func (r *PostgresRepository) Publish(ctx context.Context, weekID uuid.UUID, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE prayer_weeks
		SET status = 'published', published_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'draft'
	`, weekID, at); err != nil {
		return fmt.Errorf("publishing prayer week: %w", err)
	}
	return nil
}

// SetCompletion marks or unmarks one point for one person. Marking twice is
// the same as marking once — the primary key says so, which is also what
// makes the offline queue safe to replay.
func (r *PostgresRepository) SetCompletion(ctx context.Context, pointID, userID uuid.UUID, done bool, at time.Time) error {
	if !done {
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM prayer_completions WHERE point_id = $1 AND user_id = $2
		`, pointID, userID); err != nil {
			return fmt.Errorf("clearing completion: %w", err)
		}
		return nil
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO prayer_completions (point_id, user_id, completed_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (point_id, user_id) DO NOTHING
	`, pointID, userID, at); err != nil {
		return fmt.Errorf("recording completion: %w", err)
	}
	return nil
}

// SetReflection stores one person's words about a week, replacing whatever
// they wrote before. An empty body removes it.
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

// The columns are nullable and the domain uses empty strings, so an unset
// scripture has to become NULL rather than an empty string in the database.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
