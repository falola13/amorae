package couples_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// dbtest shares one database across packages, so values must be unique per run.
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

	// Must roll back the couple row inserted before the member insert failed.
	var n int
	if err := db.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM couples WHERE created_by = $1`, a.ID).Scan(&n); err != nil {
		t.Fatalf("count couples: %v", err)
	}
	if n != 1 {
		t.Errorf("couples created by A = %d, want 1", n)
	}
}

// pair creates two users and a live couple they both belong to.
func pair(t *testing.T, db *database.DB) (repo *couples.PostgresRepository, a, b uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	users := user.NewPostgresRepository(db)
	repo = couples.NewPostgresRepository(db)

	first, err := user.New(uniqueEmail(), "Ada", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := users.Create(ctx, first); err != nil {
		t.Fatalf("create first user: %v", err)
	}
	second, err := user.New(uniqueEmail(), "Ben", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := users.Create(ctx, second); err != nil {
		t.Fatalf("create second user: %v", err)
	}

	c, err := couples.New("", first.ID, "UTC", nil, now)
	if err != nil {
		t.Fatalf("couples.New: %v", err)
	}
	code := uniqueCode()
	if _, err := repo.Create(ctx, code, now.Add(7*24*time.Hour), c); err != nil {
		t.Fatalf("create couple: %v", err)
	}
	if err := repo.Join(ctx, second.ID, code, now); err != nil {
		t.Fatalf("join couple: %v", err)
	}
	return repo, first.ID, second.ID
}

func TestPostgresRepository_Dissolve(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo, ada, ben := pair(t, db)
	at := time.Now().UTC().Truncate(time.Microsecond)

	if err := repo.Dissolve(ctx, ada, at); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	t.Run("it ends the couple for the partner too, not just the leaver", func(t *testing.T) {
		for who, userID := range map[string]uuid.UUID{"the leaver": ada, "their partner": ben} {
			if _, err := repo.GetForUser(ctx, userID, at); !errors.Is(err, couples.ErrNotFound) {
				t.Errorf("%s still has a live couple: err = %v", who, err)
			}
		}
	})

	t.Run("both can still read what it was", func(t *testing.T) {
		for who, userID := range map[string]uuid.UUID{"the leaver": ada, "their partner": ben} {
			archived, err := repo.GetArchivedForUser(ctx, userID, at)
			if err != nil {
				t.Fatalf("GetArchivedForUser(%s): %v", who, err)
			}
			if len(archived) != 1 {
				t.Fatalf("%s sees %d ended couples, want 1", who, len(archived))
			}
			if got := archived[0].Couple.DissolvedAt; got == nil || !got.Equal(at) {
				t.Errorf("%s sees it ending at %v, want %s", who, got, at)
			}
			if len(archived[0].Members) != 2 {
				t.Errorf("%s sees %d people in it, want 2", who, len(archived[0].Members))
			}
		}
	})

	t.Run("neither is locked out of starting again", func(t *testing.T) {
		// Q-24: ending a couple must not cost someone a month of the app.
		next, err := couples.New("", ada, "UTC", nil, at)
		if err != nil {
			t.Fatalf("couples.New: %v", err)
		}
		if _, err := repo.Create(ctx, uniqueCode(), at.Add(7*24*time.Hour), next); err != nil {
			t.Fatalf("the leaver could not start a new couple: %v", err)
		}
		mine, err := repo.GetForUser(ctx, ada, at)
		if err != nil {
			t.Fatalf("GetForUser after starting again: %v", err)
		}
		if mine.Couple.ID != next.ID {
			t.Error("the live couple is not the new one")
		}
		archived, err := repo.GetArchivedForUser(ctx, ada, at)
		if err != nil || len(archived) != 1 {
			t.Errorf("the old couple went missing: %d, %v", len(archived), err)
		}
	})

	t.Run("leaving a couple you are no longer in is not found", func(t *testing.T) {
		if err := repo.Dissolve(ctx, ben, at.Add(time.Hour)); !errors.Is(err, couples.ErrNotFound) {
			t.Errorf("Dissolve: err = %v, want ErrNotFound", err)
		}
	})
}

func TestPostgresRepository_Dissolve_RevokesTheInviteStillInTheWild(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	users := user.NewPostgresRepository(db)
	repo := couples.NewPostgresRepository(db)

	ada, err := user.New(uniqueEmail(), "Ada", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := users.Create(ctx, ada); err != nil {
		t.Fatalf("create user: %v", err)
	}
	stranger, err := user.New(uniqueEmail(), "Cee", "hash", now)
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	if _, err := users.Create(ctx, stranger); err != nil {
		t.Fatalf("create user: %v", err)
	}

	c, err := couples.New("", ada.ID, "UTC", nil, now)
	if err != nil {
		t.Fatalf("couples.New: %v", err)
	}
	code := uniqueCode()
	if _, err := repo.Create(ctx, code, now.Add(7*24*time.Hour), c); err != nil {
		t.Fatalf("create couple: %v", err)
	}
	if err := repo.Dissolve(ctx, ada.ID, now); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	if err := repo.Join(ctx, stranger.ID, code, now); !errors.Is(err, couples.ErrInviteRevoked) {
		t.Fatalf("Join with a revoked code: err = %v, want ErrInviteRevoked", err)
	}

	// Even an invite forced back to pending must not open an ended couple.
	if _, err := db.Q(ctx).Exec(ctx,
		`UPDATE couple_invitations SET status = 'pending' WHERE code = $1`, code); err != nil {
		t.Fatalf("reopening the invite: %v", err)
	}
	if err := repo.Join(ctx, stranger.ID, code, now); !errors.Is(err, couples.ErrInviteInvalid) {
		t.Fatalf("Join into an ended couple: err = %v, want ErrInviteInvalid", err)
	}
}

func TestPostgresRepository_PurgeDissolvedBefore(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo, ada, ben := pair(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	endedAt := now.Add(-couples.RetentionWindow)
	if err := repo.Dissolve(ctx, ada, endedAt); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	t.Run("a window still open is left alone", func(t *testing.T) {
		purged, err := repo.PurgeDissolvedBefore(ctx, endedAt.Add(-time.Second))
		if err != nil {
			t.Fatalf("PurgeDissolvedBefore: %v", err)
		}
		if purged != 0 {
			t.Errorf("purged %d couples whose window was still open", purged)
		}
	})

	t.Run("a closed window stops being readable even before the sweep", func(t *testing.T) {
		archived, err := repo.GetArchivedForUser(ctx, ada, now.Add(time.Second))
		if err != nil {
			t.Fatalf("GetArchivedForUser: %v", err)
		}
		if len(archived) != 0 {
			t.Errorf("a couple past its window is still readable")
		}
	})

	t.Run("a closed window takes the couple and everything under it", func(t *testing.T) {
		purged, err := repo.PurgeDissolvedBefore(ctx, endedAt)
		if err != nil {
			t.Fatalf("PurgeDissolvedBefore: %v", err)
		}
		if purged != 1 {
			t.Fatalf("purged %d couples, want 1", purged)
		}

		var members, invites int
		if err := db.Q(ctx).QueryRow(ctx,
			`SELECT count(*) FROM couple_members WHERE user_id = ANY($1)`,
			[]uuid.UUID{ada, ben}).Scan(&members); err != nil {
			t.Fatalf("count members: %v", err)
		}
		if members != 0 {
			t.Errorf("%d memberships survived the purge", members)
		}
		if err := db.Q(ctx).QueryRow(ctx,
			`SELECT count(*) FROM couple_invitations WHERE created_by = ANY($1)`,
			[]uuid.UUID{ada, ben}).Scan(&invites); err != nil {
			t.Fatalf("count invitations: %v", err)
		}
		if invites != 0 {
			t.Errorf("%d invitations survived the purge", invites)
		}
	})
}

func TestPostgresRepository_UpdateCouples_Timezone(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo, ada, _ := pair(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	mine, err := repo.GetForUser(ctx, ada, now)
	if err != nil {
		t.Fatalf("GetForUser: %v", err)
	}
	if mine.Couple.Timezone != "UTC" {
		t.Fatalf("starting timezone = %q, want UTC", mine.Couple.Timezone)
	}

	lagos := "Africa/Lagos"
	if err := repo.UpdateCouples(ctx, mine.Couple.ID, nil, nil, &lagos); err != nil {
		t.Fatalf("UpdateCouples: %v", err)
	}

	t.Run("it moves for both partners at once", func(t *testing.T) {
		_, ben := mustMembers(t, db, mine.Couple.ID)
		for who, userID := range map[string]uuid.UUID{"the one who changed it": ada, "their partner": ben} {
			got, err := repo.GetForUser(ctx, userID, now)
			if err != nil {
				t.Fatalf("GetForUser(%s): %v", who, err)
			}
			if got.Couple.Timezone != lagos {
				t.Errorf("%s sees %q, want %q", who, got.Couple.Timezone, lagos)
			}
		}
	})

	t.Run("a nil timezone leaves it alone", func(t *testing.T) {
		name := "Renamed"
		if err := repo.UpdateCouples(ctx, mine.Couple.ID, nil, &name, nil); err != nil {
			t.Fatalf("UpdateCouples: %v", err)
		}
		got, err := repo.GetForUser(ctx, ada, now)
		if err != nil {
			t.Fatalf("GetForUser: %v", err)
		}
		if got.Couple.Timezone != lagos {
			t.Errorf("timezone = %q after renaming, want %q", got.Couple.Timezone, lagos)
		}
	})
}

// mustMembers returns member ids in join order.
func mustMembers(t *testing.T, db *database.DB, coupleID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	rows, err := db.Q(context.Background()).Query(context.Background(),
		`SELECT user_id FROM couple_members WHERE couple_id = $1 ORDER BY joined_at ASC`, coupleID)
	if err != nil {
		t.Fatalf("listing members: %v", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scanning member: %v", err)
		}
		ids = append(ids, id)
	}
	if len(ids) != 2 {
		t.Fatalf("couple has %d members, want 2", len(ids))
	}
	return ids[0], ids[1]
}
