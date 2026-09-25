package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// These use the real users table (via dbtest) purely as a place to insert a
// row; InTx itself doesn't care what table its caller writes to.

func TestDB_InTx_RollsBackOnError(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	id := uuid.New()
	wantErr := errors.New("boom")

	err := db.InTx(ctx, func(ctx context.Context) error {
		_, execErr := db.Q(ctx).Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash)
			VALUES ($1, $2, $3, $4)
		`, id, "rollback-"+id.String()+"@example.com", "Name", "hash")
		if execErr != nil {
			return execErr
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("InTx() error = %v, want %v", err, wantErr)
	}

	var count int
	if err := db.Q(ctx).QueryRow(ctx, `SELECT count(*) FROM users WHERE id = $1`, id).Scan(&count); err != nil {
		t.Fatalf("querying for the rolled-back row: %v", err)
	}
	if count != 0 {
		t.Errorf("found %d rows after rollback, want 0", count)
	}
}

func TestDB_InTx_ReusesOuterTransaction(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	id := uuid.New()

	err := db.InTx(ctx, func(ctx context.Context) error {
		// Nested InTx must join the outer transaction, not start a second one.
		return db.InTx(ctx, func(ctx context.Context) error {
			_, execErr := db.Q(ctx).Exec(ctx, `
				INSERT INTO users (id, email, display_name, password_hash)
				VALUES ($1, $2, $3, $4)
			`, id, "nested-"+id.String()+"@example.com", "Name", "hash")
			return execErr
		})
	})
	if err != nil {
		t.Fatalf("InTx() returned an error: %v", err)
	}

	var count int
	if err := db.Q(ctx).QueryRow(ctx, `SELECT count(*) FROM users WHERE id = $1`, id).Scan(&count); err != nil {
		t.Fatalf("querying for the committed row: %v", err)
	}
	if count != 1 {
		t.Errorf("found %d rows after commit, want 1", count)
	}
}
