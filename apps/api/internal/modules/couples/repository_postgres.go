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

func (r *PostgresRepository) Create(ctx context.Context, inviteCode string, creatorName string, c COUPLES) (COUPLES, error) {

	err := r.db.InTx(ctx, func(ctx context.Context) error {
		_, err := r.db.Q(ctx).Exec(ctx, `
	INSERT INTO couples (id, name, relationship_start_date, created_by,created_at,updated_at )
	VALUES ($1, $2, $3, $4, $5,$6)`, c.ID, c.Name, c.RelationshipStartDate, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
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
	VALUES ($1, $2, $3, $4, $5)`, inviteID, c.ID, inviteCode, c.CreatedBy, time.Now().Add(7*24*time.Hour))
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

func (r *PostgresRepository) Join(ctx context.Context, userID uuid.UUID, code string, at time.Time) (COUPLES, error) {
	var joined COUPLES
	err := r.db.InTx(ctx, func(ctx context.Context) error {
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

		var start *time.Time
		if err := r.db.Q(ctx).QueryRow(ctx, `
			SELECT id, COALESCE(name, ''), relationship_start_date, created_by, created_at, updated_at
			FROM couples WHERE id = $1
		`, coupleID).Scan(
			&joined.ID, &joined.Name, &start,
			&joined.CreatedBy, &joined.CreatedAt, &joined.UpdatedAt,
		); err != nil {
			return fmt.Errorf("loading couple: %w", err)
		}
		if start != nil {
			joined.RelationshipStartDate = *start
		}
		return nil
	})
	if err != nil {
		return COUPLES{}, err
	}
	return joined, nil
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
