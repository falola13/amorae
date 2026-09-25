package challenges

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Current loads the challenge, its days, and both partners' marks in three queries.
func (r *PostgresRepository) Current(ctx context.Context, coupleID uuid.UUID) (Challenge, error) {
	var c Challenge
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id, couple_id, template, title, started_on
		FROM challenges WHERE couple_id = $1
	`, coupleID).Scan(&c.ID, &c.CoupleID, &c.Template, &c.Title, &c.StartedOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, ErrNotFound
	}
	if err != nil {
		return Challenge{}, fmt.Errorf("loading challenge: %w", err)
	}

	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, n, prompt FROM challenge_days WHERE challenge_id = $1 ORDER BY n
	`, c.ID)
	if err != nil {
		return Challenge{}, fmt.Errorf("loading challenge days: %w", err)
	}
	at := map[uuid.UUID]int{}
	for rows.Next() {
		var d Day
		if err := rows.Scan(&d.ID, &d.N, &d.Prompt); err != nil {
			rows.Close()
			return Challenge{}, fmt.Errorf("scanning challenge day: %w", err)
		}
		d.Marks = map[uuid.UUID]Mark{}
		at[d.ID] = len(c.Days)
		c.Days = append(c.Days, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Challenge{}, fmt.Errorf("loading challenge days: %w", err)
	}

	marks, err := r.db.Q(ctx).Query(ctx, `
		SELECT p.day_id, p.user_id, p.mark
		FROM challenge_progress p
		JOIN challenge_days d ON d.id = p.day_id
		WHERE d.challenge_id = $1
	`, c.ID)
	if err != nil {
		return Challenge{}, fmt.Errorf("loading challenge progress: %w", err)
	}
	defer marks.Close()

	for marks.Next() {
		var dayID, userID uuid.UUID
		var mark Mark
		if err := marks.Scan(&dayID, &userID, &mark); err != nil {
			return Challenge{}, fmt.Errorf("scanning challenge progress: %w", err)
		}
		c.Days[at[dayID]].Marks[userID] = mark
	}
	if err := marks.Err(); err != nil {
		return Challenge{}, fmt.Errorf("loading challenge progress: %w", err)
	}
	return c, nil
}

// Start writes the challenge and days together; the couple_id unique index
// enforces one at a time (racing taps produce one challenge, not two).
func (r *PostgresRepository) Start(ctx context.Context, coupleID uuid.UUID, t Template, on time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating challenge id: %w", err)
	}

	err = r.db.InTx(ctx, func(ctx context.Context) error {
		// started_on uses the couple's local timezone, not the UTC instant,
		// so a midnight-ish start lands on the right day.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO challenges (id, couple_id, template, title, started_on, created_at, updated_at)
			SELECT $1, c.id, $3, $4, ($5 AT TIME ZONE c.timezone)::date, $5, $5
			FROM couples c
			WHERE c.id = $2
		`, id, coupleID, t.Key, t.Title, on); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrAlreadyRunning
			}
			return fmt.Errorf("starting challenge: %w", err)
		}

		for i, prompt := range t.Prompts {
			dayID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generating day id: %w", err)
			}
			if _, err := r.db.Q(ctx).Exec(ctx, `
				INSERT INTO challenge_days (id, challenge_id, n, prompt) VALUES ($1, $2, $3, $4)
			`, dayID, id, i+1, prompt); err != nil {
				return fmt.Errorf("adding challenge day: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

// SetMark upserts one partner's mark; the partner's own row is never touched (DEC-30).
func (r *PostgresRepository) SetMark(ctx context.Context, coupleID, userID uuid.UUID, n int, mark Mark, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO challenge_progress (day_id, user_id, mark, marked_at)
		SELECT d.id, $2, $4, $5
		FROM challenge_days d
		JOIN challenges c ON c.id = d.challenge_id
		WHERE c.couple_id = $1 AND d.n = $3
		ON CONFLICT (day_id, user_id) DO UPDATE SET mark = EXCLUDED.mark, marked_at = EXCLUDED.marked_at
	`, coupleID, userID, n, string(mark), at)
	if err != nil {
		return fmt.Errorf("marking challenge day: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUnknownDay
	}
	return nil
}

func (r *PostgresRepository) ClearMark(ctx context.Context, coupleID, userID uuid.UUID, n int) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM challenge_progress p
		USING challenge_days d, challenges c
		WHERE p.day_id = d.id AND d.challenge_id = c.id
		  AND c.couple_id = $1 AND p.user_id = $2 AND d.n = $3
	`, coupleID, userID, n)
	if err != nil {
		return fmt.Errorf("clearing challenge day: %w", err)
	}
	// Nothing to clear is not a failure: they were already un-marked.
	return nil
}

func (r *PostgresRepository) Leave(ctx context.Context, coupleID uuid.UUID) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM challenges WHERE couple_id = $1`, coupleID)
	if err != nil {
		return fmt.Errorf("leaving challenge: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
