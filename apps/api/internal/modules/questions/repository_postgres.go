package questions

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

// Timezone is the zone the couple's calendar runs in.
func (r *PostgresRepository) Timezone(ctx context.Context, coupleID uuid.UUID) (string, error) {
	var zone string
	err := r.db.Q(ctx).QueryRow(ctx, `SELECT timezone FROM couples WHERE id = $1`, coupleID).Scan(&zone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("couple not found: %w", err)
		}
		return "", fmt.Errorf("loading the couple's timezone: %w", err)
	}
	return zone, nil
}

const answerColumns = `id, couple_id, author_id, asked_on, question, text, updated_at`

func scanAnswers(rows pgx.Rows) ([]Answer, error) {
	defer rows.Close()
	out := []Answer{}
	for rows.Next() {
		var a Answer
		if err := rows.Scan(&a.ID, &a.CoupleID, &a.AuthorID, &a.AskedOn, &a.Question, &a.Text, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning an answer: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading answers: %w", err)
	}
	return out, nil
}

// AnswersOn is both partners' answers for one day, at most one each.
func (r *PostgresRepository) AnswersOn(ctx context.Context, coupleID uuid.UUID, day time.Time) ([]Answer, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT `+answerColumns+`
		FROM daily_answers
		WHERE couple_id = $1 AND asked_on = $2
	`, coupleID, day)
	if err != nil {
		return nil, fmt.Errorf("reading today's answers: %w", err)
	}
	return scanAnswers(rows)
}

// Keep writes this person's answer for the day, or rewrites its text. The
// question stays as it was first asked, and created_at does not move, so a
// rewrite is never announced twice.
func (r *PostgresRepository) Keep(ctx context.Context, a Answer, at time.Time) (Answer, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Answer{}, fmt.Errorf("making an answer id: %w", err)
	}
	if err := r.db.Q(ctx).QueryRow(ctx, `
		INSERT INTO daily_answers (id, couple_id, author_id, asked_on, question, text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (couple_id, author_id, asked_on)
		DO UPDATE SET text = EXCLUDED.text, updated_at = EXCLUDED.updated_at
		RETURNING id, updated_at
	`, id, a.CoupleID, a.AuthorID, a.AskedOn, a.Question, a.Text, at).Scan(&a.ID, &a.UpdatedAt); err != nil {
		return Answer{}, fmt.Errorf("keeping an answer: %w", err)
	}
	return a, nil
}

// Before is every answer on the newest `days` days that have one, earlier
// than `before`. Whether any of it may be read is the service's call.
func (r *PostgresRepository) Before(ctx context.Context, coupleID uuid.UUID, before time.Time, days int) ([]Answer, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		WITH recent AS (
			SELECT DISTINCT asked_on FROM daily_answers
			WHERE couple_id = $1 AND asked_on < $2
			ORDER BY asked_on DESC
			LIMIT $3
		)
		SELECT `+answerColumns+`
		FROM daily_answers
		WHERE couple_id = $1 AND asked_on IN (SELECT asked_on FROM recent)
		ORDER BY asked_on DESC, created_at
	`, coupleID, before, days)
	if err != nil {
		return nil, fmt.Errorf("reading past answers: %w", err)
	}
	return scanAnswers(rows)
}
