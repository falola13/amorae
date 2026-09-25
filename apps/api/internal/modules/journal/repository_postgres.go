package journal

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

// Every method takes a couple id, never a user id (DEC-19); both partners read everything.

const columns = `id, couple_id, author_id, date, tag, text`

func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Entry, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT `+columns+`
		FROM journal_entries
		WHERE couple_id = $1
		ORDER BY date DESC, created_at DESC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading the journal: %w", err)
	}
	defer rows.Close()

	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.CoupleID, &e.AuthorID, &e.Date, &e.Tag, &e.Text); err != nil {
			return nil, fmt.Errorf("scanning a journal entry: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading the journal: %w", err)
	}
	return out, nil
}

// Create computes the date in the couple's own timezone (from the couple
// row), not Go's UTC — avoids filing a late-night entry under the wrong day.
func (r *PostgresRepository) Create(ctx context.Context, e Entry, at time.Time) (Entry, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Entry{}, fmt.Errorf("generating entry id: %w", err)
	}
	e.ID = id
	if err := r.db.Q(ctx).QueryRow(ctx, `
		INSERT INTO journal_entries (id, couple_id, author_id, date, tag, text, created_at)
		SELECT $1, c.id, $3, ($6 AT TIME ZONE c.timezone)::date, $4, $5, $6
		FROM couples c
		WHERE c.id = $2
		RETURNING date
	`, id, e.CoupleID, e.AuthorID, e.Tag, e.Text, at).Scan(&e.Date); err != nil {
		return Entry{}, fmt.Errorf("writing a journal entry: %w", err)
	}
	return e, nil
}
