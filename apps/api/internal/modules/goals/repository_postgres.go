package goals

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

// Every method takes a couple id, never a user id; a goal outside it is simply not found (DEC-19).

func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Goal, error) {
	return r.load(ctx, `WHERE g.couple_id = $1`, coupleID)
}

func (r *PostgresRepository) ByID(ctx context.Context, coupleID, goalID uuid.UUID) (Goal, error) {
	found, err := r.load(ctx, `WHERE g.couple_id = $1 AND g.id = $2`, coupleID, goalID)
	if err != nil {
		return Goal{}, err
	}
	if len(found) == 0 {
		return Goal{}, ErrNotFound
	}
	return found[0], nil
}

// load reads goals and their progress in two queries however many come back.
func (r *PostgresRepository) load(ctx context.Context, where string, args ...any) ([]Goal, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT g.id, g.couple_id, g.title, COALESCE(g.why, ''), g.target, g.unit,
		       COALESCE(g.unit_label, ''), g.start_date, g.end_date, g.done
		FROM goals g `+where+`
		ORDER BY g.done, g.end_date, g.created_at
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("loading goals: %w", err)
	}

	var out []Goal
	at := map[uuid.UUID]int{}
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.CoupleID, &g.Title, &g.Why, &g.Target, &g.Unit,
			&g.UnitLabel, &g.StartDate, &g.EndDate, &g.Done); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning goal: %w", err)
		}
		at[g.ID] = len(out)
		out = append(out, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading goals: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(out))
	for _, g := range out {
		ids = append(ids, g.ID)
	}

	progress, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, goal_id, user_id, amount, date
		FROM goal_progress WHERE goal_id = ANY($1)
		ORDER BY goal_id, date, logged_at
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("loading progress: %w", err)
	}
	defer progress.Close()

	for progress.Next() {
		var p Progress
		var goalID uuid.UUID
		if err := progress.Scan(&p.ID, &goalID, &p.UserID, &p.Amount, &p.Date); err != nil {
			return nil, fmt.Errorf("scanning progress: %w", err)
		}
		i := at[goalID]
		out[i].Progress = append(out[i].Progress, p)
	}
	if err := progress.Err(); err != nil {
		return nil, fmt.Errorf("loading progress: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) Create(ctx context.Context, g Goal, at time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating goal id: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO goals (id, couple_id, title, why, target, unit, unit_label, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	`, id, g.CoupleID, g.Title, nullIfEmpty(g.Why), g.Target, string(g.Unit),
		nullIfEmpty(g.UnitLabel), g.StartDate, g.EndDate, at); err != nil {
		return uuid.UUID{}, fmt.Errorf("creating goal: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) Update(ctx context.Context, g Goal, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE goals
		SET title = $3, why = $4, target = $5, unit = $6, unit_label = $7,
		    start_date = $8, end_date = $9, done = $10, updated_at = $11
		WHERE id = $1 AND couple_id = $2
	`, g.ID, g.CoupleID, g.Title, nullIfEmpty(g.Why), g.Target, string(g.Unit),
		nullIfEmpty(g.UnitLabel), g.StartDate, g.EndDate, g.Done, at)
	if err != nil {
		return fmt.Errorf("updating goal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddProgress is scoped through the goal to the couple; an id from another goal finds nothing to add to.
func (r *PostgresRepository) AddProgress(ctx context.Context, coupleID, goalID, userID uuid.UUID, amount int64, on time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generating progress id: %w", err)
	}
	tag, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO goal_progress (id, goal_id, user_id, amount, date, logged_at)
		SELECT $1, g.id, $4, $5, $6, $7 FROM goals g
		WHERE g.id = $2 AND g.couple_id = $3
	`, id, goalID, coupleID, userID, amount, on, on)
	if err != nil {
		return fmt.Errorf("logging progress: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
