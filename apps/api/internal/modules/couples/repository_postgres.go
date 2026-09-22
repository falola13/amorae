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
			return err
		}

		inviteID, err := uuid.NewV7()
		if err != nil {
			return err
		}

		_, err = r.db.Q(ctx).Exec(ctx, `
	INSERT INTO couple_invitations (id, couple_id,code,created_by,expires_at)
	VALUES ($1, $2, $3, $4, $5)`, inviteID, c.ID, inviteCode, c.CreatedBy, inviteExpiresAt)
		if err != nil {
			return err
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

		_, err = r.db.Q(ctx).Exec(ctx, `SELECT id FROM couples WHERE id = $1 FOR UPDATE`, coupleID)
		if err != nil {
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

func translateMemberWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "couple_members_user_id_key", "couple_members_couple_user_key":
			return ErrAlreadyPaired
		}
	}
	return fmt.Errorf("adding member: %w", err)
}

func (r *PostgresRepository) GetForUser(ctx context.Context, userID uuid.UUID) (Mine, error) {
	var c COUPLES
	var start *time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT c.id, COALESCE(c.name, ''), c.timezone, c.relationship_start_date, c.created_by, c.created_at, c.updated_at
		FROM couple_members m
		JOIN couples c ON c.id = m.couple_id
		WHERE m.user_id = $1
	`, userID).Scan(&c.ID, &c.Name, &c.Timezone, &start, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Mine{}, ErrNotFound
		}
		return Mine{}, fmt.Errorf("loading couple: %w", err)
	}
	if start != nil {
		c.RelationshipStartDate = *start
	}

	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT user_id, role, onboarding_install, onboarding_notifications
		FROM couple_members WHERE couple_id = $1 ORDER BY joined_at ASC
	`, c.ID)
	if err != nil {
		return Mine{}, fmt.Errorf("listing members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Role, &m.Onboarding.Install, &m.Onboarding.Notifications); err != nil {
			return Mine{}, fmt.Errorf("scanning member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return Mine{}, fmt.Errorf("listing members: %w", err)
	}

	var code string
	err = r.db.Q(ctx).QueryRow(ctx, `
		SELECT code FROM couple_invitations WHERE couple_id = $1 ORDER BY created_at DESC LIMIT 1
	`, c.ID).Scan(&code)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Mine{}, fmt.Errorf("loading invite: %w", err)
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
