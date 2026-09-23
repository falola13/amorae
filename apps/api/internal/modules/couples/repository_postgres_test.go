package couples_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// dbtest shares one database across packages, so emails and invite codes
// must be unique per run or tests collide on unique constraints.
func uniqueEmail() string { return "test+" + uuid.NewString() + "@example.com" }
func uniqueCode() string  { return "Z" + uuid.NewString()[:5] }

func TestPostgresRepository_Create_AlreadyPaired(t *testing.T) {
	db := dbtest.New(t)
	users := user.NewPostgresRepository(db)
	repo := couples.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	a, err := user.New(uniqueEmail(), "Ada", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := users.Create(ctx, a); err != nil {
		t.Fatalf("create user: %v", err)
	}

	first, err := couples.New("", a.ID, "UTC", nil, now)
	if err != nil {
		t.Fatalf("couples.New: %v", err)
	}
	if _, err := repo.Create(ctx, uniqueCode(), now.Add(7*24*time.Hour), first); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	second, err := couples.New("", a.ID, "UTC", nil, now)
	if err != nil {
		t.Fatalf("couples.New: %v", err)
	}
	_, err = repo.Create(ctx, uniqueCode(), now.Add(7*24*time.Hour), second)
	if !errors.Is(err, couples.ErrAlreadyPaired) {
		t.Fatalf("second Create: err = %v, want ErrAlreadyPaired", err)
	}

	// The transaction must roll back the couple row inserted before the
	// member insert failed, so no orphan couple is left behind.
	var n int
	if err := db.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM couples WHERE created_by = $1`, a.ID).Scan(&n); err != nil {
		t.Fatalf("count couples: %v", err)
	}
	if n != 1 {
		t.Errorf("couples created by A = %d, want 1", n)
	}
}
