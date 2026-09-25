// Package idempotency remembers what a client's key was answered with, so a
// write sent twice does not happen twice. The rules live in
// platform/middleware; this is only the storage behind them.
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/middleware"
)

type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Claim inserts the key or loses to whoever already has it — the same
// insert-wins pattern the notification worker uses to send exactly once.
// Losing is not an error; it is the answer.
func (s *PostgresStore) Claim(
	ctx context.Context, userID uuid.UUID, key, method, path string, at time.Time,
) (bool, error) {
	tag, err := s.db.Q(ctx).Exec(ctx, `
		INSERT INTO idempotency_keys (user_id, key, method, path, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, key) DO NOTHING
	`, userID, key, method, path, at)
	if err != nil {
		return false, fmt.Errorf("claiming idempotency key: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// Lookup reads back what a key was answered with. done is false while the
// request that claimed it is still running — status is null until Save.
func (s *PostgresStore) Lookup(
	ctx context.Context, userID uuid.UUID, key string,
) (middleware.Recorded, bool, error) {
	var rec middleware.Recorded
	var status *int16
	var body []byte
	err := s.db.Q(ctx).QueryRow(ctx, `
		SELECT method, path, status, body FROM idempotency_keys
		WHERE user_id = $1 AND key = $2
	`, userID, key).Scan(&rec.Method, &rec.Path, &status, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		// Claimed and then released, most likely. Nothing stored is not an
		// error; the caller treats it as a key nobody holds.
		return middleware.Recorded{}, false, nil
	}
	if err != nil {
		return middleware.Recorded{}, false, fmt.Errorf("reading idempotency key: %w", err)
	}
	if status == nil {
		return rec, false, nil
	}
	rec.Status, rec.Body = int(*status), body
	return rec, true, nil
}

func (s *PostgresStore) Save(
	ctx context.Context, userID uuid.UUID, key string, status int, body []byte, _ time.Time,
) error {
	_, err := s.db.Q(ctx).Exec(ctx, `
		UPDATE idempotency_keys SET status = $3, body = $4
		WHERE user_id = $1 AND key = $2
	`, userID, key, int16(status), body)
	if err != nil {
		return fmt.Errorf("storing idempotent reply: %w", err)
	}
	return nil
}

func (s *PostgresStore) Release(ctx context.Context, userID uuid.UUID, key string) error {
	_, err := s.db.Q(ctx).Exec(ctx, `
		DELETE FROM idempotency_keys WHERE user_id = $1 AND key = $2
	`, userID, key)
	if err != nil {
		return fmt.Errorf("releasing idempotency key: %w", err)
	}
	return nil
}

// Prune drops keys too old to be answering a queued write. The offline queue
// keeps writes for seven days, so anything older cannot be replayed.
func (s *PostgresStore) Prune(ctx context.Context, before time.Time) (int64, error) {
	tag, err := s.db.Q(ctx).Exec(ctx, `
		DELETE FROM idempotency_keys WHERE created_at < $1
	`, before)
	if err != nil {
		return 0, fmt.Errorf("pruning idempotency keys: %w", err)
	}
	return tag.RowsAffected(), nil
}
