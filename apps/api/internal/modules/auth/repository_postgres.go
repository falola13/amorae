package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

// PostgresSessionRepository is the only implementation of SessionRepository.
// user.PostgresRepository (a different type, in a different package) plays
// the equivalent role for auth's UserRepository interface.
type PostgresSessionRepository struct {
	db *database.DB
}

func NewPostgresSessionRepository(db *database.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, s Session) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`, s.TokenHash, s.UserID, s.CreatedAt, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetByTokenHash(ctx context.Context, hash []byte) (Session, error) {
	var s Session
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT token_hash, user_id, created_at, expires_at
		FROM sessions WHERE token_hash = $1
	`, hash).Scan(&s.TokenHash, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, errSessionNotFound
		}
		return Session{}, fmt.Errorf("querying session: %w", err)
	}
	return s, nil
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, hash []byte) error {
	_, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	if err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
