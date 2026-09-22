package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// uniqueEmail keeps concurrently running test packages from colliding on
// the users_email_key constraint — dbtest.New shares one real database
// across every package's tests rather than truncating between them.
func uniqueEmail(t *testing.T) string {
	t.Helper()
	return "test+" + uuid.NewString() + "@example.com"
}

func TestPostgresRepository_CreateGetByEmailGetByID(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	u, err := user.New(uniqueEmail(t), "Ada Lovelace", "hashed-password", now)
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}

	created, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	byEmail, err := repo.GetByEmail(ctx, created.Email)
	if err != nil {
		t.Fatalf("GetByEmail() returned an error: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetByEmail() returned id %v, want %v", byEmail.ID, created.ID)
	}

	byID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned an error: %v", err)
	}
	if byID.Email != created.Email {
		t.Errorf("GetByID() returned email %q, want %q", byID.Email, created.Email)
	}
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.New())

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindNotFound {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestPostgresRepository_Create_DuplicateEmail(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()
	email := uniqueEmail(t)

	first, err := user.New(email, "First", "hashed", time.Now())
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}
	if _, err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create() returned an error: %v", err)
	}

	second, err := user.New(email, "Second", "hashed", time.Now())
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}

	_, err = repo.Create(ctx, second)
	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("Create() with a duplicate email = %v, want ErrEmailTaken", err)
	}
}

func TestPostgresRepository_Update(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()

	u, err := user.New(uniqueEmail(t), "Original Name", "hashed", time.Now())
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}
	created, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	created.DisplayName = "Updated Name"
	created.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)

	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}
	if updated.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %q, want Updated Name", updated.DisplayName)
	}

	reloaded, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned an error: %v", err)
	}
	if reloaded.DisplayName != "Updated Name" {
		t.Errorf("reloaded DisplayName = %q, want Updated Name", reloaded.DisplayName)
	}
}

func TestPostgresRepository_UpdateEmailAndTimezone(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	a, _ := user.New(uniqueEmail(t), "Ada", "hash", now)
	b, _ := user.New(uniqueEmail(t), "Bo", "hash", now)
	for _, u := range []user.User{a, b} {
		if _, err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	fresh := uniqueEmail(t)
	got, err := repo.UpdateEmail(ctx, a.ID, fresh, now)
	if err != nil || got.Email != fresh {
		t.Fatalf("UpdateEmail: got %q, err %v", got.Email, err)
	}
	if _, err := repo.UpdateEmail(ctx, a.ID, b.Email, now); err != user.ErrEmailTaken {
		t.Fatalf("UpdateEmail to another account's email: err = %v, want ErrEmailTaken", err)
	}
	if _, err := repo.UpdateEmail(ctx, uuid.New(), uniqueEmail(t), now); err != user.ErrNotFound {
		t.Fatalf("UpdateEmail unknown id: err = %v, want ErrNotFound", err)
	}

	if got.Timezone != user.DefaultTimezone {
		t.Errorf("new user timezone = %q, want %q", got.Timezone, user.DefaultTimezone)
	}
	got.Timezone = "Africa/Lagos"
	if _, err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reread, _ := repo.GetByID(ctx, a.ID)
	if reread.Timezone != "Africa/Lagos" {
		t.Errorf("timezone after Update = %q, want Africa/Lagos", reread.Timezone)
	}
}
