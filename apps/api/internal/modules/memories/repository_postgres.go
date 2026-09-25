package memories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Every method takes a couple id, never a user id: another couple's memory is
// simply not found (DEC-19).

const columns = `id, couple_id, title, date, COALESCE(location, ''), COALESCE(note, ''), COALESCE(photo_id, ''), updated_at`

// List is the couple's memories, newest first — the order an archive is read
// in, and the order the screen wants before it groups them by month.
func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Memory, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT `+columns+`
		FROM memories
		WHERE couple_id = $1
		ORDER BY date DESC, created_at DESC
	`, coupleID)
	if err != nil {
		return nil, fmt.Errorf("loading memories: %w", err)
	}
	defer rows.Close()

	out := []Memory{}
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.CoupleID, &m.Title, &m.Date,
			&m.Location, &m.Note, &m.PhotoID, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning memory: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading memories: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ByID(ctx context.Context, coupleID, id uuid.UUID) (Memory, error) {
	var m Memory
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT `+columns+`
		FROM memories
		WHERE couple_id = $1 AND id = $2
	`, coupleID, id).Scan(&m.ID, &m.CoupleID, &m.Title, &m.Date,
		&m.Location, &m.Note, &m.PhotoID, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Memory{}, ErrNotFound
		}
		return Memory{}, fmt.Errorf("loading memory: %w", err)
	}
	return m, nil
}

// Create keeps a moment. No photo: one is attached afterwards, once it has
// actually been stored somewhere (SetPhoto).
func (r *PostgresRepository) Create(ctx context.Context, m Memory, at time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating memory id: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO memories (id, couple_id, title, date, location, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
	`, id, m.CoupleID, m.Title, m.Date,
		nullIfEmpty(m.Location), nullIfEmpty(m.Note), at); err != nil {
		return uuid.UUID{}, fmt.Errorf("keeping memory: %w", err)
	}
	return id, nil
}

// SetPhoto records where a memory's picture is, or clears it. The id is the
// server's own (photos.PublicID), never a client's.
func (r *PostgresRepository) SetPhoto(ctx context.Context, coupleID, id uuid.UUID, photoID string, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE memories SET photo_id = $3, updated_at = $4
		WHERE couple_id = $1 AND id = $2
	`, coupleID, id, nullIfEmpty(photoID), at)
	if err != nil {
		return fmt.Errorf("attaching photo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, coupleID, id uuid.UUID) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM memories WHERE couple_id = $1 AND id = $2
	`, coupleID, id)
	if err != nil {
		return fmt.Errorf("deleting memory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
