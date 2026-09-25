package appreciation

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

// Every method takes a couple id, never a user id (DEC-19); both partners read all notes.

const columns = `id, couple_id, from_id, date, text, created_at`

func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Appreciation, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT `+columns+`
		FROM appreciations
		WHERE couple_id = $1
		ORDER BY created_at DESC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading notes: %w", err)
	}
	defer rows.Close()

	out := []Appreciation{}
	for rows.Next() {
		var a Appreciation
		if err := rows.Scan(&a.ID, &a.CoupleID, &a.FromID, &a.Date, &a.Text, &a.SentAt); err != nil {
			return nil, fmt.Errorf("scanning a note: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading notes: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ByID(ctx context.Context, coupleID, id uuid.UUID) (Appreciation, error) {
	var a Appreciation
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT `+columns+`
		FROM appreciations
		WHERE couple_id = $1 AND id = $2
	`, coupleID, id).Scan(&a.ID, &a.CoupleID, &a.FromID, &a.Date, &a.Text, &a.SentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Appreciation{}, ErrNotFound
		}
		return Appreciation{}, fmt.Errorf("loading a note: %w", err)
	}
	return a, nil
}

// Create files the note under the couple's local day (from the couple row,
// not Go's UTC) — see journal repository.
func (r *PostgresRepository) Create(ctx context.Context, a Appreciation, at time.Time) (Appreciation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Appreciation{}, fmt.Errorf("generating note id: %w", err)
	}
	a.ID, a.SentAt = id, at
	if err := r.db.Q(ctx).QueryRow(ctx, `
		INSERT INTO appreciations (id, couple_id, from_id, date, text, created_at)
		SELECT $1, c.id, $3, ($5 AT TIME ZONE c.timezone)::date, $4, $5
		FROM couples c
		WHERE c.id = $2
		RETURNING date
	`, id, a.CoupleID, a.FromID, a.Text, at).Scan(&a.Date); err != nil {
		return Appreciation{}, fmt.Errorf("sending a note: %w", err)
	}
	return a, nil
}

// Delete is scoped to the couple; whether this person may is decided by the caller (CanUndo).
func (r *PostgresRepository) Delete(ctx context.Context, coupleID, id uuid.UUID) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM appreciations WHERE couple_id = $1 AND id = $2
	`, coupleID, id); err != nil {
		return fmt.Errorf("taking a note back: %w", err)
	}
	return nil
}
