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

// PostgresRepository is the one concrete store behind two different
// consumer-declared interfaces: user.Service's Repository (GetByID,
// Update) and auth.Service's UserRepository (Create, GetByEmail). ISP
// means each consumer sees only the methods it uses — it doesn't mean two
// separate structs backing the same table.
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

// UpdateEmail changes only the email. A clash with another account comes
// back as ErrEmailTaken (translateWriteErr), the same as at registration.
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
	// The id came from an authenticated session, so this should always match
	// a row — but if the user was deleted between authentication and this
	// call, say so plainly instead of silently returning stale data.
	if tag.RowsAffected() == 0 {
		return User{}, ErrNotFound
	}
	return u, nil
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

// translateWriteErr turns the one constraint this table can violate — the
// email uniqueness index — into the domain error callers check for, so
// nothing above this file ever needs to know a Postgres error code.
func translateWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return ErrEmailTaken
	}
	return fmt.Errorf("writing user: %w", err)
}
