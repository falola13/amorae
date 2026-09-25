package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresSessionRepository struct {
	db *database.DB
}

func NewPostgresSessionRepository(db *database.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, s Session) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, created_at, expires_at, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, s.TokenHash, s.UserID, s.CreatedAt, s.ExpiresAt, nullIfEmpty(s.UserAgent))
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetByTokenHash(ctx context.Context, hash []byte) (Session, error) {
	var s Session
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT token_hash, user_id, created_at, expires_at, last_used_at
		FROM sessions WHERE token_hash = $1
	`, hash).Scan(&s.TokenHash, &s.UserID, &s.CreatedAt, &s.ExpiresAt, &s.LastUsedAt)
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

// ListByUser returns the user's live sessions, newest first. Expired rows are
// left out rather than shown as dead entries: they can no longer authenticate.
func (r *PostgresSessionRepository) ListByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]Session, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT token_hash, user_id, created_at, expires_at, last_used_at, coalesce(user_agent, '')
		FROM sessions
		WHERE user_id = $1 AND expires_at > $2
		ORDER BY created_at DESC
	`, userID, now)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.TokenHash, &s.UserID, &s.CreatedAt, &s.ExpiresAt, &s.LastUsedAt, &s.UserAgent); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return sessions, nil
}

func (r *PostgresSessionRepository) DeleteOthers(ctx context.Context, userID uuid.UUID, keepHash []byte) (int, error) {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2
	`, userID, keepHash)
	if err != nil {
		return 0, fmt.Errorf("deleting other sessions: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// Used by password reset: whoever asked may not be the one holding the sessions.
func (r *PostgresSessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting sessions: %w", err)
	}
	return nil
}

// TouchLastUsed records that a session was used, but only if the stored value
// is older than staleBefore — otherwise an active session would mean a write
// on every single request.
func (r *PostgresSessionRepository) TouchLastUsed(ctx context.Context, hash []byte, at, staleBefore time.Time) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE sessions SET last_used_at = $2
		WHERE token_hash = $1 AND (last_used_at IS NULL OR last_used_at < $3)
	`, hash, at, staleBefore)
	if err != nil {
		return fmt.Errorf("touching session: %w", err)
	}
	return nil
}

type PostgresPasswordResetRepository struct {
	db *database.DB
}

func NewPostgresPasswordResetRepository(db *database.DB) *PostgresPasswordResetRepository {
	return &PostgresPasswordResetRepository{db: db}
}

func (r *PostgresPasswordResetRepository) Create(ctx context.Context, hash []byte, userID uuid.UUID, expiresAt, at time.Time) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO password_resets (token_hash, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`, hash, userID, expiresAt, at)
	if err != nil {
		return fmt.Errorf("creating password reset: %w", err)
	}
	return nil
}

// Consume marks a live reset as used and returns its user. The row is locked
// so two requests racing with the same link can't both succeed.
func (r *PostgresPasswordResetRepository) Consume(ctx context.Context, hash []byte, at time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT user_id FROM password_resets
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2
		FOR UPDATE
	`, hash, at).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, errResetInvalid
		}
		return uuid.UUID{}, fmt.Errorf("finding password reset: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE password_resets SET used_at = $2 WHERE token_hash = $1
	`, hash, at); err != nil {
		return uuid.UUID{}, fmt.Errorf("using password reset: %w", err)
	}
	return userID, nil
}

type PostgresConsentRepository struct {
	db *database.DB
}

func NewPostgresConsentRepository(db *database.DB) *PostgresConsentRepository {
	return &PostgresConsentRepository{db: db}
}

func (r *PostgresConsentRepository) Record(ctx context.Context, c Consent) error {
	_, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO user_consents (id, user_id, kind, policy_version, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, c.ID, c.UserID, c.Kind, c.PolicyVersion, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("recording consent: %w", err)
	}
	return nil
}

func (r *PostgresConsentRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]Consent, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, user_id, kind, policy_version, created_at
		FROM user_consents WHERE user_id = $1 ORDER BY created_at, kind
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing consents: %w", err)
	}
	defer rows.Close()

	consents := []Consent{}
	for rows.Next() {
		var c Consent
		if err := rows.Scan(&c.ID, &c.UserID, &c.Kind, &c.PolicyVersion, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning consent: %w", err)
		}
		consents = append(consents, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing consents: %w", err)
	}
	return consents, nil
}

// nullIfEmpty keeps "unknown user agent" as NULL rather than an empty string,
// so the column means one thing.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
