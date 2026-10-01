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

// The couple's newest challenge, active ones first — so with several going,
// the most recently started of them.
const latestQuery = `
	SELECT ch.id, ch.couple_id, ch.template, ch.title, ch.status, ch.started_on,
	       ($2 AT TIME ZONE c.timezone)::date, ch.ended_at, ch.created_by
	FROM challenges ch
	JOIN couples c ON c.id = ch.couple_id
	WHERE ch.couple_id = $1
	ORDER BY (ch.status = 'active') DESC, ch.created_at DESC, ch.id DESC
	LIMIT 1
`

const getQuery = `
	SELECT ch.id, ch.couple_id, ch.template, ch.title, ch.status, ch.started_on,
	       ($3 AT TIME ZONE c.timezone)::date, ch.ended_at, ch.created_by
	FROM challenges ch
	JOIN couples c ON c.id = ch.couple_id
	WHERE ch.couple_id = $1 AND ch.id = $2
`

func (r *PostgresRepository) Latest(ctx context.Context, coupleID uuid.UUID, now time.Time) (Challenge, error) {
	c, err := r.load(ctx, latestQuery, coupleID, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, ErrNotFound
	}
	return c, err
}

// Active reads each of the couple's active challenges, oldest started first.
// At most MaxActive, so one read apiece is cheap enough.
func (r *PostgresRepository) Active(ctx context.Context, coupleID uuid.UUID, now time.Time) ([]Challenge, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id FROM challenges WHERE couple_id = $1 AND status = 'active' ORDER BY created_at, id
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("listing active challenges: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning an active challenge: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing active challenges: %w", err)
	}

	out := make([]Challenge, 0, len(ids))
	for _, id := range ids {
		c, err := r.Get(ctx, coupleID, id, now)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *PostgresRepository) Get(ctx context.Context, coupleID, id uuid.UUID, now time.Time) (Challenge, error) {
	c, err := r.load(ctx, getQuery, coupleID, id, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, ErrNoSuchChallenge
	}
	return c, err
}

// load reads the challenge, its days, both partners' marks and notes, and
// any reflections in four queries. pgx.ErrNoRows comes back as is when the
// first finds nothing, for the caller to name.
func (r *PostgresRepository) load(ctx context.Context, query string, args ...any) (Challenge, error) {
	var c Challenge
	var createdBy *uuid.UUID
	err := r.db.Q(ctx).QueryRow(ctx, query, args...).Scan(
		&c.ID, &c.CoupleID, &c.Template, &c.Title, &c.Status, &c.StartedOn, &c.Today, &c.EndedAt, &createdBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, err
	}
	if err != nil {
		return Challenge{}, fmt.Errorf("loading challenge: %w", err)
	}
	if createdBy != nil {
		c.CreatedBy = *createdBy
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
		d.Notes = map[uuid.UUID]string{}
		at[d.ID] = len(c.Days)
		c.Days = append(c.Days, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Challenge{}, fmt.Errorf("loading challenge days: %w", err)
	}

	marks, err := r.db.Q(ctx).Query(ctx, `
		SELECT p.day_id, p.user_id, p.mark, p.note
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
		var mark *Mark
		var note string
		if err := marks.Scan(&dayID, &userID, &mark, &note); err != nil {
			return Challenge{}, fmt.Errorf("scanning challenge progress: %w", err)
		}
		if mark != nil {
			c.Days[at[dayID]].Marks[userID] = *mark
		}
		if note != "" {
			c.Days[at[dayID]].Notes[userID] = note
		}
	}
	if err := marks.Err(); err != nil {
		return Challenge{}, fmt.Errorf("loading challenge progress: %w", err)
	}
	marks.Close()

	c.Reflections = map[uuid.UUID]string{}
	refs, err := r.db.Q(ctx).Query(ctx, `
		SELECT user_id, body FROM challenge_reflections WHERE challenge_id = $1
	`, c.ID)
	if err != nil {
		return Challenge{}, fmt.Errorf("loading challenge reflections: %w", err)
	}
	defer refs.Close()
	for refs.Next() {
		var userID uuid.UUID
		var body string
		if err := refs.Scan(&userID, &body); err != nil {
			return Challenge{}, fmt.Errorf("scanning challenge reflection: %w", err)
		}
		c.Reflections[userID] = body
	}
	if err := refs.Err(); err != nil {
		return Challenge{}, fmt.Errorf("loading challenge reflections: %w", err)
	}
	return c, nil
}

// Past counts "done" from each side without needing to know who the partner
// is: it is anyone else's, and a couple is two people.
func (r *PostgresRepository) Past(ctx context.Context, coupleID, userID uuid.UUID) ([]Summary, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT ch.id, ch.template, ch.title, ch.status, ch.started_on, ch.ended_at,
		       (SELECT count(*) FROM challenge_days d WHERE d.challenge_id = ch.id)::int,
		       (SELECT count(*) FROM challenge_progress p
		          JOIN challenge_days d ON d.id = p.day_id
		         WHERE d.challenge_id = ch.id AND p.user_id = $2 AND p.mark = 'done')::int,
		       (SELECT count(*) FROM challenge_progress p
		          JOIN challenge_days d ON d.id = p.day_id
		         WHERE d.challenge_id = ch.id AND p.user_id <> $2 AND p.mark = 'done')::int
		FROM challenges ch
		WHERE ch.couple_id = $1 AND ch.status <> 'active'
		ORDER BY ch.ended_at DESC NULLS LAST, ch.id DESC
	`, coupleID, userID)
	if err != nil {
		return nil, fmt.Errorf("loading past challenges: %w", err)
	}
	defer rows.Close()

	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.Template, &s.Title, &s.Status, &s.StartedOn, &s.EndedAt,
			&s.Days, &s.MyDone, &s.PartnerDone); err != nil {
			return nil, fmt.Errorf("scanning a past challenge: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading past challenges: %w", err)
	}
	return out, nil
}

// Start writes the challenge and days together. The couple's row is locked
// for the transaction so two starts at once count the same actives in turn
// and cannot both slip past MaxActive; the partial unique index on active
// templates is what keeps the same curated one from running twice.
func (r *PostgresRepository) Start(ctx context.Context, coupleID, createdBy uuid.UUID, t Template, on time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating challenge id: %w", err)
	}

	err = r.db.InTx(ctx, func(ctx context.Context) error {
		// NO KEY UPDATE conflicts with itself but not with the key-share lock the
		// insert below takes on the same row.
		var locked uuid.UUID
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT id FROM couples WHERE id = $1 FOR NO KEY UPDATE
		`, coupleID).Scan(&locked); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("locking couple: %w", err)
		}
		var active int
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT count(*)::int FROM challenges WHERE couple_id = $1 AND status = 'active'
		`, coupleID).Scan(&active); err != nil {
			return fmt.Errorf("counting active challenges: %w", err)
		}
		if active >= MaxActive {
			return ErrTooMany
		}

		// started_on uses the couple's local timezone, not the UTC instant,
		// so a midnight-ish start lands on the right day.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO challenges (id, couple_id, template, title, started_on, created_by, created_at, updated_at)
			SELECT $1, c.id, $3, $4, ($5 AT TIME ZONE c.timezone)::date, $6, $5, $5
			FROM couples c
			WHERE c.id = $2
		`, id, coupleID, t.Key, t.Title, on, createdBy); err != nil {
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

// Record upserts one partner's entry; the partner's own row is never touched
// (DEC-30). The challenge is locked first so two people marking the last days
// at once cannot both miss that the other finished it.
func (r *PostgresRepository) Record(ctx context.Context, coupleID, userID, id uuid.UUID, n int, e Entry, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		var challengeID uuid.UUID
		err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT id FROM challenges WHERE id = $1 AND couple_id = $2 AND status = 'active' FOR UPDATE
		`, id, coupleID).Scan(&challengeID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOver
		}
		if err != nil {
			return fmt.Errorf("locking challenge: %w", err)
		}

		var dayID uuid.UUID
		err = r.db.Q(ctx).QueryRow(ctx, `
			SELECT id FROM challenge_days WHERE challenge_id = $1 AND n = $2
		`, challengeID, n).Scan(&dayID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnknownDay
		}
		if err != nil {
			return fmt.Errorf("finding challenge day: %w", err)
		}

		var mark *string
		if e.SetMark {
			m := string(e.Mark)
			mark = &m
		}
		changesMark := e.SetMark || e.ClearMark
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO challenge_progress (day_id, user_id, mark, marked_at, note)
			VALUES ($1, $2, $3::challenge_mark, $4, COALESCE($5::text, ''))
			ON CONFLICT (day_id, user_id) DO UPDATE SET
				mark      = CASE WHEN $6::boolean THEN EXCLUDED.mark ELSE challenge_progress.mark END,
				marked_at = CASE WHEN $6::boolean AND EXCLUDED.mark IS NOT NULL
				                 THEN EXCLUDED.marked_at ELSE challenge_progress.marked_at END,
				note      = COALESCE($5::text, challenge_progress.note)
		`, dayID, userID, mark, at, e.Note, changesMark); err != nil {
			return fmt.Errorf("saving challenge day: %w", err)
		}
		// Neither a mark nor a note left: there is nothing to keep a row for.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM challenge_progress
			WHERE day_id = $1 AND user_id = $2 AND mark IS NULL AND note = ''
		`, dayID, userID); err != nil {
			return fmt.Errorf("clearing challenge day: %w", err)
		}

		if !e.SetMark {
			return nil
		}
		// Finished once every current member has marked every day, done or
		// skipped. Decided here, in the same write as the mark that could
		// have completed it, so it is never a beat behind.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE challenges ch
			SET status = 'finished', ended_at = $2, updated_at = $2
			WHERE ch.id = $1 AND ch.status = 'active'
			  AND EXISTS (SELECT 1 FROM couple_members m
			               WHERE m.couple_id = ch.couple_id AND m.ended_at IS NULL)
			  AND NOT EXISTS (
			        SELECT 1
			        FROM couple_members m
			        JOIN challenge_days d ON d.challenge_id = ch.id
			        WHERE m.couple_id = ch.couple_id AND m.ended_at IS NULL
			          AND NOT EXISTS (
			                SELECT 1 FROM challenge_progress p
			                 WHERE p.day_id = d.id AND p.user_id = m.user_id
			                   AND p.mark IS NOT NULL)
			      )
		`, challengeID, at); err != nil {
			return fmt.Errorf("finishing challenge: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) End(ctx context.Context, coupleID, id uuid.UUID, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE challenges SET status = 'ended', ended_at = $2, updated_at = $2
		WHERE id = $1 AND couple_id = $3 AND status = 'active'
	`, id, at, coupleID)
	if err != nil {
		return fmt.Errorf("leaving challenge: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetReflection(ctx context.Context, coupleID, userID, id uuid.UUID, text string, at time.Time) error {
	if text == "" {
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM challenge_reflections f
			USING challenges ch
			WHERE f.challenge_id = ch.id AND ch.couple_id = $1 AND ch.id = $2 AND f.user_id = $3
		`, coupleID, id, userID); err != nil {
			return fmt.Errorf("removing reflection: %w", err)
		}
		// Nothing to remove is not a failure: they had not written one.
		return nil
	}
	tag, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO challenge_reflections (challenge_id, user_id, body, created_at, updated_at)
		SELECT ch.id, $3, $4, $5, $5
		FROM challenges ch
		WHERE ch.id = $2 AND ch.couple_id = $1 AND ch.status <> 'active'
		ON CONFLICT (challenge_id, user_id) DO UPDATE SET body = EXCLUDED.body, updated_at = EXCLUDED.updated_at
	`, coupleID, id, userID, text, at)
	if err != nil {
		return fmt.Errorf("saving reflection: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoSuchChallenge
	}
	return nil
}
