package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/auth"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

func TestPostgresSessionRepository_CreateGetDelete(t *testing.T) {
	db := dbtest.New(t)
	userRepo := user.NewPostgresRepository(db)
	sessionRepo := auth.NewPostgresSessionRepository(db)
	ctx := context.Background()

	// A session's user_id has a foreign key into users, so it needs a real
	// user row to point at.
	u, err := user.New("session-test-"+uuid.NewString()+"@example.com", "Name", "hash", time.Now())
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}
	createdUser, err := userRepo.Create(ctx, u)
	if err != nil {
		t.Fatalf("creating the test user: %v", err)
	}

	session := auth.Session{
		TokenHash: []byte("integration-test-hash-" + uuid.NewString()),
		UserID:    createdUser.ID,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond),
	}

	if err := sessionRepo.Create(ctx, session); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	got, err := sessionRepo.GetByTokenHash(ctx, session.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() returned an error: %v", err)
	}
	if got.UserID != createdUser.ID {
		t.Errorf("UserID = %v, want %v", got.UserID, createdUser.ID)
	}

	if err := sessionRepo.Delete(ctx, session.TokenHash); err != nil {
		t.Fatalf("Delete() returned an error: %v", err)
	}

	if _, err := sessionRepo.GetByTokenHash(ctx, session.TokenHash); err == nil {
		t.Error("GetByTokenHash() after Delete() returned nil error, want not-found")
	}
}

func TestPostgresSessionRepository_Delete_IsIdempotent(t *testing.T) {
	db := dbtest.New(t)
	sessionRepo := auth.NewPostgresSessionRepository(db)
	ctx := context.Background()

	hash := []byte("never-created-" + uuid.NewString())

	if err := sessionRepo.Delete(ctx, hash); err != nil {
		t.Fatalf("Delete() of a nonexistent session returned an error: %v", err)
	}
}
