package anniversary

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

// Couple is when they started and the zone their calendar runs in.
func (r *PostgresRepository) Couple(ctx context.Context, coupleID uuid.UUID) (*time.Time, string, error) {
	var start *time.Time
	var timezone string
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT relationship_start_date, timezone FROM couples WHERE id = $1
	`, coupleID).Scan(&start, &timezone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNoAnniversary
		}
		return nil, "", fmt.Errorf("loading the couple: %w", err)
	}
	return start, timezone, nil
}

// Letters is every letter either of them has written, newest anniversary
// first. Whether one may be read is the service's call, not this query's.
func (r *PostgresRepository) Letters(ctx context.Context, coupleID uuid.UUID) ([]Letter, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, couple_id, author_id, opens_on, text, updated_at
		FROM anniversary_letters
		WHERE couple_id = $1
		ORDER BY opens_on DESC, created_at
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading letters: %w", err)
	}
	defer rows.Close()

	out := []Letter{}
	for rows.Next() {
		var l Letter
		if err := rows.Scan(&l.ID, &l.CoupleID, &l.AuthorID, &l.OpensOn, &l.Text, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning a letter: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading letters: %w", err)
	}
	return out, nil
}

// Keep writes this person's letter for that anniversary, or replaces it.
func (r *PostgresRepository) Keep(ctx context.Context, l Letter, at time.Time) (Letter, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Letter{}, fmt.Errorf("generating letter id: %w", err)
	}
	if err := r.db.Q(ctx).QueryRow(ctx, `
		INSERT INTO anniversary_letters (id, couple_id, author_id, opens_on, text, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (couple_id, author_id, opens_on)
		DO UPDATE SET text = EXCLUDED.text, updated_at = EXCLUDED.updated_at
		RETURNING id, updated_at
	`, id, l.CoupleID, l.AuthorID, l.OpensOn, l.Text, at).Scan(&l.ID, &l.UpdatedAt); err != nil {
		return Letter{}, fmt.Errorf("keeping a letter: %w", err)
	}
	return l, nil
}

// Year counts what the couple did on days from `from` up to, not including,
// `to`. `today` keeps a plan that is still ahead from counting as one that
// happened. Dates are the couple's own (DEC-27), the same reading the
// timeline gives each source.
func (r *PostgresRepository) Year(ctx context.Context, coupleID uuid.UUID, from, to, today time.Time) (Year, error) {
	y := Year{From: from, To: to}
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM prayer_completions pc
		     JOIN prayer_points p ON p.id = pc.point_id
		     JOIN prayer_weeks w ON w.id = p.week_id
		    WHERE w.couple_id = $1 AND pc.prayed_on >= $2 AND pc.prayed_on < $3),
		  (SELECT count(*) FROM prayer_points p
		     JOIN prayer_weeks w ON w.id = p.week_id
		    WHERE w.couple_id = $1 AND p.answered_at IS NOT NULL
		      AND (p.answered_at AT TIME ZONE c.timezone)::date >= $2
		      AND (p.answered_at AT TIME ZONE c.timezone)::date < $3),
		  (SELECT count(*) FROM prayer_weeks w
		    WHERE w.couple_id = $1 AND w.status = 'published'
		      AND w.week_start >= $2 AND w.week_start < $3),
		  (SELECT count(*) FROM memories m
		    WHERE m.couple_id = $1 AND m.date >= $2 AND m.date < $3),
		  (SELECT count(*) FROM events e
		    WHERE e.couple_id = $1 AND e.date >= $2 AND e.date < $3
		      AND (e.done OR (NOT e.didnt_happen AND e.date < $4))),
		  (SELECT count(*) FROM challenges ch
		    WHERE ch.couple_id = $1 AND ch.status = 'finished' AND ch.ended_at IS NOT NULL
		      AND (ch.ended_at AT TIME ZONE c.timezone)::date >= $2
		      AND (ch.ended_at AT TIME ZONE c.timezone)::date < $3),
		  (SELECT count(*) FROM appreciations a
		    WHERE a.couple_id = $1 AND a.date >= $2 AND a.date < $3),
		  (SELECT count(*) FROM journal_entries j
		    WHERE j.couple_id = $1 AND j.date >= $2 AND j.date < $3)
		FROM couples c
		WHERE c.id = $1
	`, coupleID, from, to, today).Scan(
		&y.Prayed, &y.Answered, &y.PrayerWeeks, &y.Memories, &y.Events,
		&y.Challenges, &y.Notes, &y.Journal,
	)
	if err != nil {
		return Year{}, fmt.Errorf("counting the year: %w", err)
	}
	return y, nil
}
