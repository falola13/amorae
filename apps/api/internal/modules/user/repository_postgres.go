package user

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

// PostgresRepository backs both user.Service's Repository and auth.Service's
// UserRepository — one table, two consumer-declared interfaces.
type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// userColumns is every column a User is read from, in scanUser's order.
const userColumns = `id, email, display_name, password_hash, timezone, created_at, updated_at, last_login_at`

func (r *PostgresRepository) Create(ctx context.Context, u User) (User, error) {
	_, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO users (id, email, display_name, password_hash, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, u.ID, u.Email, u.DisplayName, u.PasswordHash, u.Timezone, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return User{}, translateWriteErr(err)
	}
	return u, nil
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	return r.scanOne(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return r.scanOne(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

// UpdateEmail changes only the email; a clash returns ErrEmailTaken, same as registration.
func (r *PostgresRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string, at time.Time) (User, error) {
	u, err := r.scanOne(ctx, `
		UPDATE users SET email = $2, updated_at = $3 WHERE id = $1
		RETURNING `+userColumns, id, email, at)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return User{}, translateWriteErr(err)
	}
	return u, err
}

func (r *PostgresRepository) Update(ctx context.Context, u User) (User, error) {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE users SET display_name = $2, timezone = $3, updated_at = $4
		WHERE id = $1
	`, u.ID, u.DisplayName, u.Timezone, u.UpdatedAt)
	if err != nil {
		return User{}, translateWriteErr(err)
	}
	// Should always match a row; ErrNotFound covers the user being deleted mid-request.
	if tag.RowsAffected() == 0 {
		return User{}, ErrNotFound
	}
	return u, nil
}
func (r *PostgresRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1
	`, id, hash, at)
	if err != nil {
		return fmt.Errorf("writing password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE users SET last_login_at = $2 WHERE id = $1
	`, id, at)
	if err != nil {
		return fmt.Errorf("writing last_login_at: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) scanOne(ctx context.Context, query string, args ...any) (User, error) {
	var u User
	err := r.db.Q(ctx).QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Timezone, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("querying user: %w", err)
	}
	return u, nil
}

// DeleteMe cascades sessions/memberships/invitations. Couples are released
// first (deleted if last member, else created_by transfers — FK doesn't
// cascade); memberships can be plural per Q-24.
func (r *PostgresRepository) DeleteMe(ctx context.Context, id uuid.UUID) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		coupleIDs, err := r.coupleIDsOf(ctx, id)
		if err != nil {
			return err
		}
		for _, coupleID := range coupleIDs {
			if err := r.releaseCouple(ctx, id, coupleID); err != nil {
				return err
			}
		}

		tag, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			return fmt.Errorf("deleting user: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (r *PostgresRepository) coupleIDsOf(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT couple_id FROM couple_members WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("finding memberships: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning membership: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding memberships: %w", err)
	}
	return ids, nil
}

func (r *PostgresRepository) releaseCouple(ctx context.Context, userID, coupleID uuid.UUID) error {
	var partnerID uuid.UUID
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT user_id FROM couple_members
		WHERE couple_id = $1 AND user_id <> $2
	`, coupleID, userID).Scan(&partnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM couples WHERE id = $1`, coupleID); err != nil {
			return fmt.Errorf("deleting couple: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("finding partner: %w", err)
	}

	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE couples SET created_by = $2, updated_at = now()
		WHERE id = $1 AND created_by = $3
	`, coupleID, partnerID, userID); err != nil {
		return fmt.Errorf("transferring couple: %w", err)
	}
	return nil
}

// translateWriteErr maps the email-uniqueness violation to ErrEmailTaken so
// callers never see a raw Postgres error code.
func translateWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return ErrEmailTaken
	}
	return fmt.Errorf("writing user: %w", err)
}
