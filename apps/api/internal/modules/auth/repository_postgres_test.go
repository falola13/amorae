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

// seedSession creates a user-owned session row and returns it.
func seedSession(t *testing.T, repo *auth.PostgresSessionRepository, userID uuid.UUID, created time.Time, expires time.Time, ua string) auth.Session {
	t.Helper()
	s := auth.Session{
		TokenHash: []byte("hash-" + uuid.NewString()),
		UserID:    userID,
		CreatedAt: created.UTC().Truncate(time.Microsecond),
		ExpiresAt: expires.UTC().Truncate(time.Microsecond),
		UserAgent: ua,
	}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("creating session: %v", err)
	}
	return s
}

func seedUser(t *testing.T, repo *user.PostgresRepository) user.User {
	t.Helper()
	u, err := user.New("session-test-"+uuid.NewString()+"@example.com", "Name", "hash", time.Now())
	if err != nil {
		t.Fatalf("user.New() returned an error: %v", err)
	}
	created, err := repo.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("creating the test user: %v", err)
	}
	return created
}

func TestPostgresSessionRepository_ListByUser_NewestFirstAndLiveOnly(t *testing.T) {
	db := dbtest.New(t)
	sessions := auth.NewPostgresSessionRepository(db)
	users := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now()

	owner := seedUser(t, users)
	stranger := seedUser(t, users)

	older := seedSession(t, sessions, owner.ID, now.Add(-48*time.Hour), now.Add(time.Hour), "Mozilla/5.0 (iPhone) Safari/604.1")
	newer := seedSession(t, sessions, owner.ID, now.Add(-1*time.Hour), now.Add(time.Hour), "")
	seedSession(t, sessions, owner.ID, now.Add(-72*time.Hour), now.Add(-time.Minute), "expired")
	seedSession(t, sessions, stranger.ID, now, now.Add(time.Hour), "someone else's")

	got, err := sessions.ListByUser(ctx, owner.ID, now)
	if err != nil {
		t.Fatalf("ListByUser() returned an error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d sessions, want 2 (expired and other users' excluded)", len(got))
	}
	if string(got[0].TokenHash) != string(newer.TokenHash) {
		t.Error("ListByUser() should return the newest session first")
	}
	if string(got[1].TokenHash) != string(older.TokenHash) {
		t.Error("ListByUser() returned the wrong second session")
	}
	if got[1].UserAgent == "" {
		t.Error("UserAgent should round-trip for a session that has one")
	}
	if got[0].UserAgent != "" {
		t.Errorf("a session stored without a user agent should read back empty, got %q", got[0].UserAgent)
	}
}

func TestPostgresSessionRepository_DeleteOthers_KeepsTheCurrentSession(t *testing.T) {
	db := dbtest.New(t)
	sessions := auth.NewPostgresSessionRepository(db)
	users := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now()

	owner := seedUser(t, users)
	stranger := seedUser(t, users)
	keep := seedSession(t, sessions, owner.ID, now, now.Add(time.Hour), "")
	gone := seedSession(t, sessions, owner.ID, now, now.Add(time.Hour), "")
	untouched := seedSession(t, sessions, stranger.ID, now, now.Add(time.Hour), "")

	ended, err := sessions.DeleteOthers(ctx, owner.ID, keep.TokenHash)
	if err != nil {
		t.Fatalf("DeleteOthers() returned an error: %v", err)
	}
	if ended != 1 {
		t.Errorf("ended = %d, want 1", ended)
	}

	if _, err := sessions.GetByTokenHash(ctx, keep.TokenHash); err != nil {
		t.Errorf("the current session must survive: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctx, gone.TokenHash); err == nil {
		t.Error("the other session should have been deleted")
	}
	if _, err := sessions.GetByTokenHash(ctx, untouched.TokenHash); err != nil {
		t.Errorf("another user's session must never be touched: %v", err)
	}
}

func TestPostgresSessionRepository_TouchLastUsed_OnlyWhenStale(t *testing.T) {
	db := dbtest.New(t)
	sessions := auth.NewPostgresSessionRepository(db)
	users := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	owner := seedUser(t, users)
	s := seedSession(t, sessions, owner.ID, now, now.Add(time.Hour), "")

	// Never used before: records it.
	if err := sessions.TouchLastUsed(ctx, s.TokenHash, now, now.Add(-time.Hour)); err != nil {
		t.Fatalf("TouchLastUsed() returned an error: %v", err)
	}
	first, err := sessions.GetByTokenHash(ctx, s.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() returned an error: %v", err)
	}
	if first.LastUsedAt == nil {
		t.Fatal("LastUsedAt is nil after the first touch")
	}

	// Within the hour: left alone.
	later := now.Add(10 * time.Minute)
	if err := sessions.TouchLastUsed(ctx, s.TokenHash, later, later.Add(-time.Hour)); err != nil {
		t.Fatalf("TouchLastUsed() returned an error: %v", err)
	}
	again, err := sessions.GetByTokenHash(ctx, s.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() returned an error: %v", err)
	}
	if !again.LastUsedAt.Equal(*first.LastUsedAt) {
		t.Errorf("LastUsedAt moved within the hour: %v → %v", first.LastUsedAt, again.LastUsedAt)
	}

	// An hour on: recorded again.
	muchLater := now.Add(2 * time.Hour)
	if err := sessions.TouchLastUsed(ctx, s.TokenHash, muchLater, muchLater.Add(-time.Hour)); err != nil {
		t.Fatalf("TouchLastUsed() returned an error: %v", err)
	}
	final, err := sessions.GetByTokenHash(ctx, s.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash() returned an error: %v", err)
	}
	if !final.LastUsedAt.After(*first.LastUsedAt) {
		t.Errorf("LastUsedAt = %v, want it to move once an hour has passed", final.LastUsedAt)
	}
}
