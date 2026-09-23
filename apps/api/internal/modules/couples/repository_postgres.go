package couples

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

func (r *PostgresRepository) Create(ctx context.Context, inviteCode string, inviteExpiresAt time.Time, c COUPLES) (COUPLES, error) {

	// The column is nullable and the field is a bare time.Time, so an unset
	// date has to become NULL here — inserting the zero value stores
	// 0001-01-01, which reads back as a real date and breaks date maths.
	var start *time.Time
	if !c.RelationshipStartDate.IsZero() {
		start = &c.RelationshipStartDate
	}

	err := r.db.InTx(ctx, func(ctx context.Context) error {
		_, err := r.db.Q(ctx).Exec(ctx, `
	INSERT INTO couples (id, name, timezone, relationship_start_date, created_by,created_at,updated_at )
	VALUES ($1, $2, $3, $4, $5, $6, $7)`, c.ID, c.Name, c.Timezone, start, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
		if err != nil {
			return err
		}

		memberID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = r.db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, memberID, c.ID, c.CreatedBy, c.CreatedAt)
		if err != nil {
			return translateMemberWriteErr(err)
		}

		inviteID, err := uuid.NewV7()
		if err != nil {
			return err
		}

		_, err = r.db.Q(ctx).Exec(ctx, `
	INSERT INTO couple_invitations (id, couple_id,code,created_by,expires_at)
	VALUES ($1, $2, $3, $4, $5)`, inviteID, c.ID, inviteCode, c.CreatedBy, inviteExpiresAt)
		if err != nil {
			return translateInviteWriteErr(err)
		}

		return nil
	})

	if err != nil {
		return COUPLES{}, err
	}

	return c, nil
}

// Join adds the caller to the invite's couple and accepts the invite in one
// transaction. It does not return the couple: the service re-reads through
// GetForUser, which is the only query that also carries members and invite.
func (r *PostgresRepository) Join(ctx context.Context, userID uuid.UUID, code string, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		var (
			inviteID  uuid.UUID
			coupleID  uuid.UUID
			status    string
			createdBy uuid.UUID
			expiresAt time.Time
		)
		err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT id, couple_id, status, created_by, expires_at
			FROM couple_invitations
			WHERE code = $1
			FOR UPDATE
		`, code).Scan(&inviteID, &coupleID, &status, &createdBy, &expiresAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInviteInvalid
			}
			return fmt.Errorf("looking up invite: %w", err)
		}

		switch status {
		case "accepted":
			return ErrInviteUsed
		case "revoked":
			return ErrInviteRevoked
		case "expired":
			return ErrInviteExpired
		case "pending":
		default:
			return ErrInviteInvalid
		}
		if !at.Before(expiresAt) {
			return ErrInviteExpired
		}
		if createdBy == userID {
			return ErrInviteInvalid
		}

		// Locks the couple, and reads the one thing that makes an otherwise
		// valid code useless: the couple it belongs to has ended. Dissolving
		// revokes pending invites, so this is the second lock on that door.
		var dissolvedAt *time.Time
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT dissolved_at FROM couples WHERE id = $1 FOR UPDATE
		`, coupleID).Scan(&dissolvedAt); err != nil {
			return fmt.Errorf("locking couple: %w", err)
		}
		if dissolvedAt != nil {
			return ErrInviteInvalid
		}

		var members int
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT COUNT(*) FROM couple_members WHERE couple_id = $1
		`, coupleID).Scan(&members); err != nil {
			return fmt.Errorf("counting members: %w", err)
		}
		if members >= 2 {
			return ErrCoupleFull
		}

		memberID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = r.db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, memberID, coupleID, userID, at)
		if err != nil {
			return translateMemberWriteErr(err)
		}

		_, err = r.db.Q(ctx).Exec(ctx, `
			UPDATE couple_invitations
			SET status = 'accepted', accepted_at = $2
			WHERE id = $1 AND status = 'pending'
		`, inviteID, at)
		if err != nil {
			return fmt.Errorf("accepting invite: %w", err)
		}

		return nil
	})
}

func (r *PostgresRepository) GetForUser(ctx context.Context, userID uuid.UUID, now time.Time) (Mine, error) {
	var c COUPLES
	var start *time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT c.id, COALESCE(c.name, ''), c.timezone, c.relationship_start_date, c.created_by, c.created_at, c.updated_at, c.dissolved_at
		FROM couple_members m
		JOIN couples c ON c.id = m.couple_id
		WHERE m.user_id = $1 AND m.ended_at IS NULL
	`, userID).Scan(&c.ID, &c.Name, &c.Timezone, &start, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.DissolvedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Mine{}, ErrNotFound
		}
		return Mine{}, fmt.Errorf("loading couple: %w", err)
	}
	if start != nil {
		c.RelationshipStartDate = *start
	}

	members, err := r.membersOf(ctx, c.ID)
	if err != nil {
		return Mine{}, err
	}

	var code string
	if len(members) < 2 {
		err = r.db.Q(ctx).QueryRow(ctx, `
			SELECT code FROM couple_invitations WHERE couple_id = $1 AND status = 'pending' AND expires_at > $2 ORDER BY created_at DESC LIMIT 1
		`, c.ID, now).Scan(&code)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return Mine{}, fmt.Errorf("loading invite: %w", err)
		}
	}

	return Mine{Couple: c, Members: members, InviteCode: code}, nil
}

func (r *PostgresRepository) UpdateCouples(ctx context.Context, coupleID uuid.UUID, start *time.Time, name *string) error {

	tag, err := r.db.Q(ctx).Exec(ctx, `
	UPDATE couples
	SET relationship_start_date = COALESCE($2, relationship_start_date) , name = COALESCE($3, name),
	updated_at = now()
	WHERE id = $1 
	`, coupleID, start, name)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// role is already trimmed and length-checked by ValidateRole, so it is
// assigned outright rather than through a COALESCE fallback.
func (r *PostgresRepository) UpdateRole(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, role string) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE couple_members 
		SET role = $3, updated_at = now()
		WHERE couple_id = $1 AND user_id = $2
		`, coupleID, id, role)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Dissolve ends the caller's couple for both partners.
//
// Three things happen together or not at all: the couple gets its end date,
// both memberships are marked ended, and any invite still pending is revoked
// so a stranger holding the code cannot walk into a couple that no longer
// exists.
//
// The memberships are kept, not deleted — they are how both partners go on
// reading and exporting until PurgeDissolvedBefore removes the couple. Ending
// them is what frees each person to start again (Q-24).
func (r *PostgresRepository) Dissolve(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		// Conditional on the couple still being live, so two partners tapping
		// "leave" at the same moment settle on one end date rather than the
		// later one restarting the other's window.
		var coupleID uuid.UUID
		err := r.db.Q(ctx).QueryRow(ctx, `
			UPDATE couples c
			SET dissolved_at = $2, dissolved_by = $1, updated_at = $2
			FROM couple_members m
			WHERE m.couple_id = c.id AND m.user_id = $1
			  AND m.ended_at IS NULL AND c.dissolved_at IS NULL
			RETURNING c.id
		`, userID, at).Scan(&coupleID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("ending couple: %w", err)
		}

		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE couple_members SET ended_at = $2
			WHERE couple_id = $1 AND ended_at IS NULL
		`, coupleID, at); err != nil {
			return fmt.Errorf("ending memberships: %w", err)
		}

		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE couple_invitations SET status = 'revoked'
			WHERE couple_id = $1 AND status = 'pending'
		`, coupleID); err != nil {
			return fmt.Errorf("revoking invite: %w", err)
		}

		return nil
	})
}

// GetArchivedForUser is the couples this person used to be in and can still
// read: ended, not yet purged, and inside the retention window.
//
// The window is applied here rather than left to the sweeper, so a sweeper
// that stops running cannot quietly turn 30 days of access into forever.
func (r *PostgresRepository) GetArchivedForUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]Mine, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT c.id, COALESCE(c.name, ''), c.timezone, c.relationship_start_date,
		       c.created_by, c.created_at, c.updated_at, c.dissolved_at
		FROM couple_members m
		JOIN couples c ON c.id = m.couple_id
		WHERE m.user_id = $1 AND m.ended_at IS NOT NULL AND c.dissolved_at > $2
		ORDER BY c.dissolved_at DESC
	`, userID, now.Add(-RetentionWindow))
	if err != nil {
		return nil, fmt.Errorf("listing ended couples: %w", err)
	}

	var archived []Mine
	for rows.Next() {
		var c COUPLES
		var start *time.Time
		if err := rows.Scan(&c.ID, &c.Name, &c.Timezone, &start, &c.CreatedBy,
			&c.CreatedAt, &c.UpdatedAt, &c.DissolvedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning ended couple: %w", err)
		}
		if start != nil {
			c.RelationshipStartDate = *start
		}
		archived = append(archived, Mine{Couple: c})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing ended couples: %w", err)
	}

	// Members come after the rows are closed: one connection, one query at a
	// time, and this is never more than a couple or two.
	for i := range archived {
		members, err := r.membersOf(ctx, archived[i].Couple.ID)
		if err != nil {
			return nil, err
		}
		archived[i].Members = members
	}
	return archived, nil
}

func (r *PostgresRepository) membersOf(ctx context.Context, coupleID uuid.UUID) ([]Member, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT user_id, role, onboarding_install, onboarding_notifications
		FROM couple_members WHERE couple_id = $1 ORDER BY joined_at ASC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Role, &m.Onboarding.Install, &m.Onboarding.Notifications); err != nil {
			return nil, fmt.Errorf("scanning member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	return members, nil
}

// purgeLockID keeps two API instances from sweeping at the same moment. The
// number is arbitrary but must stay fixed: it only has to differ from every
// other advisory lock this codebase takes.
const purgeLockID = 20260923

// PurgeDissolvedBefore deletes couples whose retention window closed at or
// before cutoff, and reports how many went. One DELETE is the whole purge:
// members, invitations and every couple-owned table cascade from this row,
// so a module added later is covered the day its table references couples.
func (r *PostgresRepository) PurgeDissolvedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var purged int64
	err := r.db.InTx(ctx, func(ctx context.Context) error {
		var mine bool
		if err := r.db.Q(ctx).QueryRow(ctx,
			`SELECT pg_try_advisory_xact_lock($1)`, purgeLockID).Scan(&mine); err != nil {
			return fmt.Errorf("taking the purge lock: %w", err)
		}
		if !mine {
			// Someone else is sweeping. There is nothing to wait for: the
			// next tick will find whatever they leave behind.
			return nil
		}

		tag, err := r.db.Q(ctx).Exec(ctx, `
			DELETE FROM couples
			WHERE dissolved_at IS NOT NULL AND dissolved_at <= $1
		`, cutoff)
		if err != nil {
			return fmt.Errorf("purging ended couples: %w", err)
		}
		purged = tag.RowsAffected()
		return nil
	})
	return purged, err
}

// Only the caller's own membership is touched: install and notifications
// describe one person's phone, so neither partner can mark them for the other.
func (r *PostgresRepository) UpdateOnboarding(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, install *bool, notifications *bool) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE couple_members
		SET onboarding_install = COALESCE($3, onboarding_install),
		    onboarding_notifications = COALESCE($4, onboarding_notifications),
		    updated_at = now()
		WHERE couple_id = $1 AND user_id = $2
		`, coupleID, id, install, notifications)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ReplaceInvite revokes the couple's pending invite and issues a new one, in
// one transaction, so there is never a moment with two live codes or none.
func (r *PostgresRepository) ReplaceInvite(ctx context.Context, userID uuid.UUID, code string, expiresAt, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		var coupleID uuid.UUID
		err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT c.id FROM couple_members m
			JOIN couples c ON c.id = m.couple_id
			WHERE m.user_id = $1
			FOR UPDATE OF c
		`, userID).Scan(&coupleID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("locking couple: %w", err)
		}

		var members int
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT COUNT(*) FROM couple_members WHERE couple_id = $1
		`, coupleID).Scan(&members); err != nil {
			return fmt.Errorf("counting members: %w", err)
		}
		if members >= 2 {
			return ErrCoupleFull
		}

		if _, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE couple_invitations SET status = 'revoked'
			WHERE couple_id = $1 AND status = 'pending'
		`, coupleID); err != nil {
			return fmt.Errorf("revoking invite: %w", err)
		}

		inviteID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = r.db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_invitations (id, couple_id, code, created_by, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, inviteID, coupleID, code, userID, expiresAt, at)
		if err != nil {
			return translateInviteWriteErr(err)
		}
		return nil
	})
}

// The code column is UNIQUE (couple_invitations_code_key, 00001_init.sql).
func translateInviteWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "couple_invitations_code_key" {
		return errCodeTaken
	}
	return fmt.Errorf("creating invite: %w", err)
}

func translateMemberWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "couple_members_live_user_key", "couple_members_couple_user_key":
			return ErrAlreadyPaired
		}
	}
	return fmt.Errorf("adding member: %w", err)
}
