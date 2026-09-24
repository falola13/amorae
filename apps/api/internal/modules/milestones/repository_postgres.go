package milestones

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

// Every method takes a couple id, never a user id: another couple's date is
// simply not found (DEC-19).

// List is the couple's dates, oldest first. The client decides what "coming
// up" means, because that depends on today and on which of them recur
// (FR-DATE-002.AC1).
func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Milestone, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, couple_id, title, date, COALESCE(sub, ''), reminder
		FROM milestones
		WHERE couple_id = $1
		ORDER BY date, created_at
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading dates: %w", err)
	}
	defer rows.Close()

	out := []Milestone{}
	for rows.Next() {
		var m Milestone
		if err := rows.Scan(&m.ID, &m.CoupleID, &m.Title, &m.Date, &m.Sub, &m.Reminder); err != nil {
			return nil, fmt.Errorf("scanning date: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading dates: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ByID(ctx context.Context, coupleID, id uuid.UUID) (Milestone, error) {
	var m Milestone
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id, couple_id, title, date, COALESCE(sub, ''), reminder
		FROM milestones
		WHERE couple_id = $1 AND id = $2
	`, coupleID, id).Scan(&m.ID, &m.CoupleID, &m.Title, &m.Date, &m.Sub, &m.Reminder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Milestone{}, ErrNotFound
		}
		return Milestone{}, fmt.Errorf("loading date: %w", err)
	}
	return m, nil
}

func (r *PostgresRepository) Create(ctx context.Context, m Milestone, at time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating date id: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO milestones (id, couple_id, title, date, sub, reminder, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
	`, id, m.CoupleID, m.Title, m.Date, nullIfEmpty(m.Sub), m.Reminder, at); err != nil {
		return uuid.UUID{}, fmt.Errorf("keeping date: %w", err)
	}
	return id, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
