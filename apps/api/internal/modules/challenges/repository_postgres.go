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
	SELECT ch.id, ch.couple_id, ch.template, ch.title, ch.status, ch.kind, ch.started_on,
	       ($2 AT TIME ZONE c.timezone)::date, ch.ended_at, ch.created_by
	FROM challenges ch
	JOIN couples c ON c.id = ch.couple_id
	WHERE ch.couple_id = $1
	ORDER BY (ch.status = 'active') DESC, ch.created_at DESC, ch.id DESC
	LIMIT 1
`

const getQuery = `
	SELECT ch.id, ch.couple_id, ch.template, ch.title, ch.status, ch.kind, ch.started_on,
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
		&c.ID, &c.CoupleID, &c.Template, &c.Title, &c.Status, &c.Kind, &c.StartedOn, &c.Today, &c.EndedAt, &createdBy)
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
		SELECT ch.id, ch.template, ch.title, ch.status, ch.kind, ch.created_by, ch.started_on, ch.ended_at,
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
		var createdBy *uuid.UUID
		if err := rows.Scan(&s.ID, &s.Template, &s.Title, &s.Status, &s.Kind, &createdBy, &s.StartedOn, &s.EndedAt,
			&s.Days, &s.MyDone, &s.PartnerDone); err != nil {
			return nil, fmt.Errorf("scanning a past challenge: %w", err)
		}
		if createdBy != nil {
			s.CreatedBy = *createdBy
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading past challenges: %w", err)
	}
	return out, nil
}

// Spec is what a new challenge is made of. A nil StartedOn means today, in
// the couple's own calendar.
type Spec struct {
	Template  Template
	Kind      Kind
	StartedOn *time.Time
}

// Today is the couple's own date as of `now`.
func (r *PostgresRepository) Today(ctx context.Context, coupleID uuid.UUID, now time.Time) (time.Time, error) {
	var today time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT ($2 AT TIME ZONE c.timezone)::date FROM couples c WHERE c.id = $1
	`, coupleID, now).Scan(&today)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("reading the couple's date: %w", err)
	}
	return today, nil
}

// activeFor is how many challenges this person has going: the ones they share
// and their own, not their partner's.
func (r *PostgresRepository) activeFor(ctx context.Context, coupleID, userID uuid.UUID) (int, error) {
	var n int
	if err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT count(*)::int FROM challenges
		WHERE couple_id = $1 AND status = 'active' AND (kind = 'together' OR created_by = $2)
	`, coupleID, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting active challenges: %w", err)
	}
	return n, nil
}

// partnerOf is the other current member of the couple and their name; the
// zero id when there is no one else.
func (r *PostgresRepository) partnerOf(ctx context.Context, coupleID, userID uuid.UUID) (uuid.UUID, string, error) {
	var id uuid.UUID
	var name string
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT u.id, u.display_name
		FROM couple_members m JOIN users u ON u.id = m.user_id
		WHERE m.couple_id = $1 AND m.ended_at IS NULL AND m.user_id <> $2
		LIMIT 1
	`, coupleID, userID).Scan(&id, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, "", nil
	}
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("finding the partner: %w", err)
	}
	return id, name, nil
}

// partnerRoom is nil when the partner has room for one more challenge that
// counts for them, and the refusal to give when they do not. A couple with no
// partner has no one to be full.
func (r *PostgresRepository) partnerRoom(ctx context.Context, coupleID, userID uuid.UUID) error {
	partner, name, err := r.partnerOf(ctx, coupleID, userID)
	if err != nil || partner == uuid.Nil {
		return err
	}
	theirs, err := r.activeFor(ctx, coupleID, partner)
	if err != nil {
		return err
	}
	if theirs >= MaxActive {
		return ErrPartnerFull(name)
	}
	return nil
}

// lockCouple serialises everything that counts a couple's challenges.
// NO KEY UPDATE conflicts with itself but not with the key-share lock an
// insert of a challenge takes on the same row.
func (r *PostgresRepository) lockCouple(ctx context.Context, coupleID uuid.UUID) error {
	var locked uuid.UUID
	if err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id FROM couples WHERE id = $1 FOR NO KEY UPDATE
	`, coupleID).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("locking couple: %w", err)
	}
	return nil
}

// Start writes the challenge and days together. The couple's row is locked
// for the transaction so two starts at once count the same actives in turn
// and cannot both slip past MaxActive. The cap is per person: the starter's
// own count must have room, and a together challenge needs room for the
// partner too. The same curated challenge cannot run twice for one person (a
// together one counts for both); the partial unique indexes are only the
// backstop for that, since an index cannot look across kinds.
func (r *PostgresRepository) Start(ctx context.Context, coupleID, createdBy uuid.UUID, s Spec, now time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating challenge id: %w", err)
	}
	t := s.Template

	err = r.db.InTx(ctx, func(ctx context.Context) error {
		if err := r.lockCouple(ctx, coupleID); err != nil {
			return err
		}
		mine, err := r.activeFor(ctx, coupleID, createdBy)
		if err != nil {
			return err
		}
		if mine >= MaxActive {
			return ErrTooMany
		}
		if s.Kind != KindMine {
			if err := r.partnerRoom(ctx, coupleID, createdBy); err != nil {
				return err
			}
		}
		if t.Key != CustomKey {
			var running bool
			if err := r.db.Q(ctx).QueryRow(ctx, `
				SELECT EXISTS (
				    SELECT 1 FROM challenges
				    WHERE couple_id = $1 AND status = 'active' AND template = $2
				      AND ($3::boolean OR kind = 'together' OR created_by = $4)
				)
			`, coupleID, t.Key, s.Kind != KindMine, createdBy).Scan(&running); err != nil {
				return fmt.Errorf("checking for the same challenge: %w", err)
			}
			if running {
				return ErrAlreadyRunning
			}
		}

		// started_on uses the couple's local timezone, not the UTC instant,
		// so a midnight-ish start lands on the right day.
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO challenges (id, couple_id, template, title, kind, started_on, created_by, created_at, updated_at)
			SELECT $1, c.id, $3, $4, $8, COALESCE($7::date, ($5 AT TIME ZONE c.timezone)::date), $6, $5, $5
			FROM couples c
			WHERE c.id = $2
		`, id, coupleID, t.Key, t.Title, now, createdBy, s.StartedOn, string(s.Kind)); err != nil {
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

// lockChallenge locks one of the couple's challenges for the rest of the
// transaction and reads it as of `now`, so what is checked is what is
// written. One that is not the couple's is ErrNoSuchChallenge; whether it is
// still active is left to the caller's rules (ErrOver).
func (r *PostgresRepository) lockChallenge(ctx context.Context, coupleID, id uuid.UUID, now time.Time) (Challenge, error) {
	var locked uuid.UUID
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT id FROM challenges WHERE id = $1 AND couple_id = $2 FOR UPDATE
	`, id, coupleID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return Challenge{}, ErrNoSuchChallenge
	}
	if err != nil {
		return Challenge{}, fmt.Errorf("locking challenge: %w", err)
	}
	return r.Get(ctx, coupleID, id, now)
}

// Edit changes a challenge's title, start day or kind. It reads the challenge
// under a lock and lets Challenge.CheckEdit decide, so a partner marking at
// the same moment cannot slip in between "nobody has joined in" and the
// change. Making it shared again needs room on the partner's side too.
func (r *PostgresRepository) Edit(ctx context.Context, coupleID, userID, id uuid.UUID, e Edit, now time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		if err := r.lockCouple(ctx, coupleID); err != nil {
			return err
		}
		c, err := r.lockChallenge(ctx, coupleID, id, now)
		if err != nil {
			return err
		}
		e, err = c.CheckEdit(userID, e)
		if err != nil {
			return err
		}
		if e.Title == nil && e.StartedOn == nil && e.Kind == nil {
			return nil
		}
		if e.Kind != nil && *e.Kind == KindTogether {
			if err := r.partnerRoom(ctx, coupleID, userID); err != nil {
				return err
			}
		}
		var kind *string
		if e.Kind != nil {
			k := string(*e.Kind)
			kind = &k
		}
		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE challenges
			SET title      = COALESCE($3, title),
			    started_on = COALESCE($4::date, started_on),
			    kind       = COALESCE($5, kind),
			    updated_at = $6
			WHERE id = $1 AND couple_id = $2
		`, id, coupleID, e.Title, e.StartedOn, kind, now); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrAlreadyRunning
			}
			return fmt.Errorf("editing challenge: %w", err)
		}
		return nil
	})
}

// ReplacePlan sets the days' texts by position: kept days keep their marks
// and notes, extra days are added, and days past the end go only when
// Challenge.CheckPlan finds nobody has written on them. Fewer days can leave
// every participant having answered every one, which finishes it here, in the
// same write, like the mark that would have.
func (r *PostgresRepository) ReplacePlan(ctx context.Context, coupleID, userID, id uuid.UUID, prompts []string, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		c, err := r.lockChallenge(ctx, coupleID, id, at)
		if err != nil {
			return err
		}
		if err := c.CheckPlan(userID, prompts); err != nil {
			return err
		}
		for i, prompt := range prompts {
			if i < len(c.Days) {
				if c.Days[i].Prompt == prompt {
					continue
				}
				if _, err := r.db.Q(ctx).Exec(ctx, `
					UPDATE challenge_days SET prompt = $2 WHERE id = $1
				`, c.Days[i].ID, prompt); err != nil {
					return fmt.Errorf("rewording challenge day: %w", err)
				}
				continue
			}
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
		if _, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM challenge_days WHERE challenge_id = $1 AND n > $2
		`, id, len(prompts)); err != nil {
			return fmt.Errorf("removing challenge days: %w", err)
		}
		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE challenges SET updated_at = $2 WHERE id = $1
		`, id, at); err != nil {
			return fmt.Errorf("touching challenge: %w", err)
		}
		return r.finishIfAnswered(ctx, id, at)
	})
}

// finishIfAnswered finishes the challenge once every participant has marked
// every day, done or skipped. A shared one is everybody still in the couple;
// a "just me" one is its creator alone. Decided in the same write as the
// change that could have completed it, so it is never a beat behind.
func (r *PostgresRepository) finishIfAnswered(ctx context.Context, id uuid.UUID, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE challenges ch
		SET status = 'finished', ended_at = $2, updated_at = $2
		WHERE ch.id = $1 AND ch.status = 'active'
		  AND EXISTS (SELECT 1 FROM couple_members m
		               WHERE m.couple_id = ch.couple_id AND m.ended_at IS NULL
		                 AND (ch.kind = 'together' OR m.user_id = ch.created_by))
		  AND NOT EXISTS (
		        SELECT 1
		        FROM couple_members m
		        JOIN challenge_days d ON d.challenge_id = ch.id
		        WHERE m.couple_id = ch.couple_id AND m.ended_at IS NULL
		          AND (ch.kind = 'together' OR m.user_id = ch.created_by)
		          AND NOT EXISTS (
		                SELECT 1 FROM challenge_progress p
		                 WHERE p.day_id = d.id AND p.user_id = m.user_id
		                   AND p.mark IS NOT NULL)
		      )
	`, id, at); err != nil {
		return fmt.Errorf("finishing challenge: %w", err)
	}
	return nil
}

// Record upserts one partner's entry; the partner's own row is never touched
// (DEC-30). The challenge is locked first so two people marking the last days
// at once cannot both miss that the other finished it. A "just me" challenge
// is its creator's alone to write on.
func (r *PostgresRepository) Record(ctx context.Context, coupleID, userID, id uuid.UUID, n int, e Entry, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		var challengeID uuid.UUID
		var kind Kind
		var createdBy *uuid.UUID
		err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT id, kind, created_by FROM challenges WHERE id = $1 AND couple_id = $2 AND status = 'active' FOR UPDATE
		`, id, coupleID).Scan(&challengeID, &kind, &createdBy)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOver
		}
		if err != nil {
			return fmt.Errorf("locking challenge: %w", err)
		}
		if kind == KindMine && (createdBy == nil || *createdBy != userID) {
			return ErrNoSuchChallenge
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
		return r.finishIfAnswered(ctx, challengeID, at)
	})
}

func (r *PostgresRepository) End(ctx context.Context, coupleID, userID, id uuid.UUID, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE challenges SET status = 'ended', ended_at = $2, updated_at = $2
		WHERE id = $1 AND couple_id = $3 AND status = 'active'
		  AND (kind = 'together' OR created_by = $4)
	`, id, at, coupleID, userID)
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
		  AND (ch.kind = 'together' OR ch.created_by = $3)
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
