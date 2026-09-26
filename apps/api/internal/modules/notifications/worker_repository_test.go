package notifications_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/notifications"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// pair inserts two users and a live couple they both belong to. Deliberately
// bare SQL, like milestones' own pair() helper: this only needs rows to
// exist, not the user/couples packages' own validation.
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

// TestPostgresRepository_ImportantDates_DerivedRows checks the part of the
// ImportantDates query that isn't in the milestones table at all: a
// couple's anniversary and its current members' birthdays, unioned in
// directly from couples and users (see worker_repository.go's own comment,
// and milestones.AnniversaryID/BirthdayID for the Go side of the same ids).
func TestPostgresRepository_ImportantDates_DerivedRows(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)

	coupleID, a, b := pair(t, db)

	start := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	if _, err := db.Q(ctx).Exec(ctx, `
		UPDATE couples SET relationship_start_date = $2 WHERE id = $1
	`, coupleID, start); err != nil {
		t.Fatalf("set relationship_start_date: %v", err)
	}
	// Ada's birthday has a year; Bo's doesn't.
	if _, err := db.Q(ctx).Exec(ctx, `
		UPDATE users SET birth_month = 9, birth_day = 30, birth_year = 1990 WHERE id = $1
	`, a); err != nil {
		t.Fatalf("set Ada's birthday: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		UPDATE users SET birth_month = 2, birth_day = 29 WHERE id = $1
	`, b); err != nil {
		t.Fatalf("set Bo's birthday: %v", err)
	}

	got, err := repo.ImportantDates(ctx)
	if err != nil {
		t.Fatalf("ImportantDates: %v", err)
	}

	var anniversaryForA, anniversaryForB bool
	var bosBirthdayToAda, bosBirthdayToBo bool
	var adasBirthdayToBo bool
	for _, c := range got {
		switch {
		case c.Title == "Our anniversary" && c.UserID == a:
			anniversaryForA = true
		case c.Title == "Our anniversary" && c.UserID == b:
			anniversaryForB = true
		case c.UserID == a && c.Date.Month() == time.February:
			bosBirthdayToAda = true
			if c.YearKnown {
				t.Error("Bo's yearless birthday reported YearKnown true")
			}
		case c.UserID == b && c.Date.Month() == time.February:
			bosBirthdayToBo = true
		case c.UserID == b && c.Date.Month() == time.September:
			adasBirthdayToBo = true
			if !c.YearKnown {
				t.Error("Ada's birthday, which has a year, reported YearKnown false")
			}
		}
	}

	if !anniversaryForA || !anniversaryForB {
		t.Errorf("anniversary reached A=%v B=%v, want both", anniversaryForA, anniversaryForB)
	}
	if !bosBirthdayToAda {
		t.Error("Bo's birthday never reached Ada")
	}
	if bosBirthdayToBo {
		t.Error("Bo's own birthday reached Bo — a reminder must never go to the birthday person themself")
	}
	if !adasBirthdayToBo {
		t.Error("Ada's birthday never reached Bo")
	}
}
