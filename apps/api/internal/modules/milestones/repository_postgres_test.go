package milestones_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/milestones"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// fixedCouple always answers CoupleFor with the couple id it was built
// with, standing in for couples.Service so these tests don't need the
// whole couples module wired up.
type fixedCouple struct{ coupleID uuid.UUID }

func (f fixedCouple) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.coupleID, nil
}

// pair inserts two users and a live couple they both belong to. Deliberately
// bare SQL rather than the user/couples packages: this test only needs rows
// to exist, and pulling in two more modules to build them isn't worth it.
func pair(t *testing.T, db *database.DB) (coupleID, a, b uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	a, b = uuid.New(), uuid.New()
	names := map[uuid.UUID]string{a: "Ada", b: "Bo"}
	for id, name := range names {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, 'hash', $4, $4)
		`, id, "test+"+uuid.NewString()+"@example.com", name, now); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	coupleID = uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & Bo', 'UTC', $2, $3, $3)
	`, coupleID, a, now); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	for id := range names {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), coupleID, id, now); err != nil {
			t.Fatalf("insert member: %v", err)
		}
	}
	return coupleID, a, b
}

func setBirthday(t *testing.T, db *database.DB, userID uuid.UUID, month, day int, year *int) {
	t.Helper()
	if _, err := db.Q(context.Background()).Exec(context.Background(), `
		UPDATE users SET birth_month = $2, birth_day = $3, birth_year = $4 WHERE id = $1
	`, userID, month, day, year); err != nil {
		t.Fatalf("set birthday: %v", err)
	}
}

func endMembership(t *testing.T, db *database.DB, coupleID, userID uuid.UUID) {
	t.Helper()
	if _, err := db.Q(context.Background()).Exec(context.Background(), `
		UPDATE couple_members SET ended_at = now() WHERE couple_id = $1 AND user_id = $2
	`, coupleID, userID); err != nil {
		t.Fatalf("end membership: %v", err)
	}
}

// TestDerivedID_MatchesSQL keeps milestones.AnniversaryID/BirthdayID —
// what List and Delete use — identical to the md5(...)::uuid expression
// notifications' worker_repository.go ImportantDates computes
// independently, so a couple's derived rows carry the same id wherever
// they're produced. Two implementations of the same id, checked against
// each other here rather than one importing the other.
func TestDerivedID_MatchesSQL(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()

	coupleID := uuid.New()
	userID := uuid.New()

	cases := []struct {
		name string
		seed string
		want uuid.UUID
	}{
		{"anniversary", coupleID.String() + ":anniversary", milestones.AnniversaryID(coupleID)},
		{"birthday", coupleID.String() + ":birthday:" + userID.String(), milestones.BirthdayID(coupleID, userID)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got uuid.UUID
			if err := db.Q(ctx).QueryRow(ctx, `SELECT md5($1)::uuid`, tc.seed).Scan(&got); err != nil {
				t.Fatalf("querying Postgres's own md5(...)::uuid: %v", err)
			}
			if got != tc.want {
				t.Errorf("SQL = %v, Go DerivedID = %v, want identical", got, tc.want)
			}
		})
	}
}

func TestPostgresRepository_Derived(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := milestones.NewPostgresRepository(db)

	t.Run("no anniversary and no birthdays is nothing derived", func(t *testing.T) {
		coupleID, _, _ := pair(t, db)
		got, err := repo.Derived(ctx, coupleID)
		if err != nil {
			t.Fatalf("Derived: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %d derived dates, want 0", len(got))
		}
	})

	t.Run("an anniversary from the couple, and birthdays from current members only", func(t *testing.T) {
		coupleID, a, b := pair(t, db)
		start := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
		if _, err := db.Q(ctx).Exec(ctx, `
			UPDATE couples SET relationship_start_date = $2 WHERE id = $1
		`, coupleID, start); err != nil {
			t.Fatalf("set relationship_start_date: %v", err)
		}
		year := 1990
		setBirthday(t, db, a, 9, 30, &year)
		setBirthday(t, db, b, 2, 29, nil) // no year: Feb 29 must still work

		got, err := repo.Derived(ctx, coupleID)
		if err != nil {
			t.Fatalf("Derived: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d derived dates, want 3 (anniversary + 2 birthdays): %+v", len(got), got)
		}

		var anniversary, aBirthday, bBirthday *milestones.Milestone
		for i := range got {
			m := &got[i]
			switch {
			case m.Source == milestones.SourceAnniversary:
				anniversary = m
			case m.Source == milestones.SourceBirthday && m.About == a:
				aBirthday = m
			case m.Source == milestones.SourceBirthday && m.About == b:
				bBirthday = m
			}
		}
		if anniversary == nil {
			t.Fatal("no anniversary in Derived()")
		}
		if anniversary.ID != milestones.AnniversaryID(coupleID) || anniversary.Title != "Our anniversary" ||
			!anniversary.Date.Equal(start) || !anniversary.Reminder || !anniversary.YearKnown {
			t.Errorf("anniversary = %+v", anniversary)
		}

		if aBirthday == nil || bBirthday == nil {
			t.Fatalf("missing a birthday: a=%v b=%v", aBirthday, bBirthday)
		}
		if aBirthday.ID != milestones.BirthdayID(coupleID, a) || !aBirthday.YearKnown ||
			aBirthday.Date.Year() != 1990 || aBirthday.Date.Month() != time.September || aBirthday.Date.Day() != 30 {
			t.Errorf("Ada's birthday = %+v", aBirthday)
		}
		if bBirthday.ID != milestones.BirthdayID(coupleID, b) || bBirthday.YearKnown ||
			bBirthday.Date.Year() != 2000 || bBirthday.Date.Month() != time.February || bBirthday.Date.Day() != 29 {
			t.Errorf("Bo's yearless Feb 29 birthday = %+v", bBirthday)
		}

		endMembership(t, db, coupleID, b)
		got, err = repo.Derived(ctx, coupleID)
		if err != nil {
			t.Fatalf("Derived after ending a membership: %v", err)
		}
		for _, m := range got {
			if m.Source == milestones.SourceBirthday && m.About == b {
				t.Error("a former member's birthday was still derived")
			}
		}
	})
}

func TestService_List_IncludesDerivedDates(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := milestones.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	setBirthday(t, db, a, 3, 1, nil)

	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO milestones (id, couple_id, title, date, reminder, created_at, updated_at)
		VALUES ($1, $2, 'Our engagement', '2022-01-01', true, now(), now())
	`, uuid.New(), coupleID); err != nil {
		t.Fatalf("insert stored milestone: %v", err)
	}

	svc := milestones.NewService(repo, fixedCouple{coupleID: coupleID}, time.Now)
	got, err := svc.List(ctx, uuid.New())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d milestones, want 2 (stored + derived birthday): %+v", len(got), got)
	}

	var sawStored, sawBirthday bool
	for _, m := range got {
		switch m.Source {
		case "":
			sawStored = true
		case milestones.SourceBirthday:
			sawBirthday = true
			if m.YearKnown {
				t.Error("a birthday with no year reported YearKnown true")
			}
		}
	}
	if !sawStored || !sawBirthday {
		t.Errorf("stored seen = %v, derived birthday seen = %v", sawStored, sawBirthday)
	}
}

func TestService_Delete_RefusesADerivedID(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := milestones.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	setBirthday(t, db, a, 3, 1, nil)

	svc := milestones.NewService(repo, fixedCouple{coupleID: coupleID}, time.Now)

	err := svc.Delete(ctx, uuid.New(), milestones.BirthdayID(coupleID, a))
	if err != milestones.ErrDerived {
		t.Fatalf("Delete(derived id) error = %v, want ErrDerived", err)
	}
}
