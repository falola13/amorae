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

// Every method takes a couple id, never a user id; another couple's date is
// simply not found (DEC-19).

// List returns the couple's dates, oldest first; the client decides "coming
// up" (depends on today and recurrence) (FR-DATE-002.AC1).
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

// Derived computes the anniversary (from couples.relationship_start_date)
// and each current member's birthday (from users.birth_month/day/year),
// reading them from where they actually live rather than copying either
// into this table (BR-DATE-01's mirror image: one kind that's never stored
// at all). Ended members are left out, same as the worker's own queries.
func (r *PostgresRepository) Derived(ctx context.Context, coupleID uuid.UUID) ([]Milestone, error) {
	out := []Milestone{}

	var start *time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT relationship_start_date FROM couples WHERE id = $1
	`, coupleID).Scan(&start)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("loading couple for derived dates: %w", err)
	}
	if start != nil {
		out = append(out, Milestone{
			ID:        AnniversaryID(coupleID),
			CoupleID:  coupleID,
			Title:     "Our anniversary",
			Date:      *start,
			Reminder:  true,
			Source:    SourceAnniversary,
			YearKnown: true,
		})
	}

	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, u.display_name, u.birth_month, u.birth_day, u.birth_year
		FROM couple_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.couple_id = $1 AND m.ended_at IS NULL AND u.birth_month IS NOT NULL
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading members for derived dates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var name string
		var month, day int
		var year *int
		if err := rows.Scan(&userID, &name, &month, &day, &year); err != nil {
			return nil, fmt.Errorf("scanning member for derived dates: %w", err)
		}
		// Year 2000 is the same placeholder ValidateBirthday uses when
		// asked without one — never a real year, only a stand-in that
		// keeps Feb 29 representable.
		calendarYear, yearKnown := 2000, year != nil
		if yearKnown {
			calendarYear = *year
		}
		out = append(out, Milestone{
			ID:        BirthdayID(coupleID, userID),
			CoupleID:  coupleID,
			Title:     name + "’s birthday",
			Date:      time.Date(calendarYear, time.Month(month), day, 0, 0, 0, 0, time.UTC),
			Reminder:  true,
			Source:    SourceBirthday,
			YearKnown: yearKnown,
			About:     userID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading members for derived dates: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, coupleID, id uuid.UUID) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM milestones WHERE couple_id = $1 AND id = $2
	`, coupleID, id)
	if err != nil {
		return fmt.Errorf("deleting date: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
