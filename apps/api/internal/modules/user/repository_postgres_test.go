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

// uniqueEmail avoids colliding on users_email_key: dbtest.New shares one
// real database across test packages rather than truncating between them.
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

func TestPostgresRepository_DeleteMe_RemovesSoleCouple(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	owner, err := user.New(uniqueEmail(t), "Ada", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := repo.Create(ctx, owner); err != nil {
		t.Fatalf("Create: %v", err)
	}

	coupleID := uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & partner', 'UTC', $2, $3, $3)
	`, coupleID, owner.ID, now); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couple_members (id, couple_id, user_id, joined_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), coupleID, owner.ID, now); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couple_invitations (id, couple_id, code, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), coupleID, "A"+uuid.NewString()[:5], owner.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("insert invitation: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`, []byte(uuid.NewString()), owner.ID, now, now.Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	if err := repo.DeleteMe(ctx, owner.ID); err != nil {
		t.Fatalf("DeleteMe: %v", err)
	}
	if _, err := repo.GetByID(ctx, owner.ID); err != user.ErrNotFound {
		t.Fatalf("GetByID after delete: err = %v, want ErrNotFound", err)
	}
	for _, q := range []struct {
		name  string
		query string
	}{
		{"couple", `SELECT count(*) FROM couples WHERE id = $1`},
		{"members", `SELECT count(*) FROM couple_members WHERE couple_id = $1`},
		{"invitations", `SELECT count(*) FROM couple_invitations WHERE couple_id = $1`},
	} {
		var n int
		if err := db.Q(ctx).QueryRow(ctx, q.query, coupleID).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", q.name, err)
		}
		if n != 0 {
			t.Errorf("%s rows = %d, want 0", q.name, n)
		}
	}
}

func TestPostgresRepository_DeleteMe_KeepsPartnersCouple(t *testing.T) {
	db := dbtest.New(t)
	repo := user.NewPostgresRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	owner, _ := user.New(uniqueEmail(t), "Ada", "hash", now)
	partner, _ := user.New(uniqueEmail(t), "Bo", "hash", now)
	for _, u := range []user.User{owner, partner} {
		if _, err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	coupleID := uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & Bo', 'UTC', $2, $3, $3)
	`, coupleID, owner.ID, now); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	for _, userID := range []uuid.UUID{owner.ID, partner.ID} {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), coupleID, userID, now); err != nil {
			t.Fatalf("insert member: %v", err)
		}
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couple_invitations (id, couple_id, code, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), coupleID, "B"+uuid.NewString()[:5], owner.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("insert invitation: %v", err)
	}

	if err := repo.DeleteMe(ctx, owner.ID); err != nil {
		t.Fatalf("DeleteMe: %v", err)
	}

	var createdBy uuid.UUID
	if err := db.Q(ctx).QueryRow(ctx, `SELECT created_by FROM couples WHERE id = $1`, coupleID).Scan(&createdBy); err != nil {
		t.Fatalf("couple should remain: %v", err)
	}
	if createdBy != partner.ID {
		t.Errorf("created_by = %v, want partner %v", createdBy, partner.ID)
	}

	var members int
	if err := db.Q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM couple_members WHERE couple_id = $1 AND user_id = $2
	`, coupleID, partner.ID).Scan(&members); err != nil {
		t.Fatalf("count partner membership: %v", err)
	}
	if members != 1 {
		t.Errorf("partner memberships = %d, want 1", members)
	}

	var invites int
	if err := db.Q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM couple_invitations WHERE created_by = $1
	`, owner.ID).Scan(&invites); err != nil {
		t.Fatalf("count invitations: %v", err)
	}
	if invites != 0 {
		t.Errorf("creator invitations = %d, want 0", invites)
	}
}
