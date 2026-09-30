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

// seedEvent writes an event dated `days` from today (the couple is UTC).
func seedEvent(t *testing.T, db *database.DB, coupleID, creator uuid.UUID, title string, days int, reminders []string, kind string, done, didntHappen bool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	date := time.Now().UTC().AddDate(0, 0, days)
	if reminders == nil {
		reminders = []string{}
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO events (id, couple_id, title, date, start_time, reminders, done, didnt_happen, created_by, kind, created_at, updated_at)
		VALUES ($1, $2, $3, $4, '19:00', $5, $6, $7, $8, $9, now(), now())
	`, id, coupleID, title, date, reminders, done, didntHappen, creator, kind); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	return id
}

func TestPostgresRepository_DueEventReminders_OnePerReminder(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)

	several := seedEvent(t, db, coupleID, a, "Dinner", 0, []string{"1 hour before", "at 16:00", "the morning of"}, "together", false, false)
	none := seedEvent(t, db, coupleID, a, "Nothing set", 0, nil, "together", false, false)
	done := seedEvent(t, db, coupleID, a, "Done", 0, []string{"1 hour before"}, "together", true, false)
	didnt := seedEvent(t, db, coupleID, a, "Didnt", 0, []string{"1 hour before"}, "together", false, true)
	mine := seedEvent(t, db, coupleID, a, "Dentist", 0, []string{"at 09:00"}, "mine", false, false)

	got, err := repo.DueEventReminders(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("DueEventReminders: %v", err)
	}

	perUser := map[uuid.UUID]map[string]int{}
	for _, c := range got {
		if perUser[c.EventID] == nil {
			perUser[c.EventID] = map[string]int{}
		}
		perUser[c.EventID][c.UserID.String()+"|"+c.Reminder]++
	}

	// Three reminders, two partners: six candidates, one for each pairing.
	if n := len(perUser[several]); n != 6 {
		t.Errorf("an event with three reminders gave %d candidates for two people, want 6: %v", n, perUser[several])
	}
	for _, r := range []string{"1 hour before", "at 16:00", "the morning of"} {
		for _, u := range []uuid.UUID{a, b} {
			if perUser[several][u.String()+"|"+r] != 1 {
				t.Errorf("no single candidate for %s / %q", u, r)
			}
		}
	}
	for name, id := range map[string]uuid.UUID{"no reminders": none, "done": done, "didnt_happen": didnt} {
		if len(perUser[id]) != 0 {
			t.Errorf("%s still came back to be reminded about: %v", name, perUser[id])
		}
	}
	// A mine event's reminder is its creator's alone.
	if len(perUser[mine]) != 1 || perUser[mine][a.String()+"|at 09:00"] != 1 {
		t.Errorf("a mine event's reminders = %v, want only its creator's", perUser[mine])
	}
}

func TestPostgresRepository_EndedEvents_SkipsAnsweredOnes(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)

	open := seedEvent(t, db, coupleID, a, "Open", -1, nil, "together", false, false)
	done := seedEvent(t, db, coupleID, a, "Done", -1, nil, "together", true, false)
	didnt := seedEvent(t, db, coupleID, a, "Didnt", -1, nil, "together", false, true)

	// Bo turned follow-ups off; Ada never touched hers.
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_preferences (user_id, event_followups) VALUES ($1, false)
	`, b); err != nil {
		t.Fatalf("set preference: %v", err)
	}

	got, err := repo.EndedEvents(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("EndedEvents: %v", err)
	}
	followups := map[uuid.UUID]map[uuid.UUID]bool{}
	for _, c := range got {
		if followups[c.EventID] == nil {
			followups[c.EventID] = map[uuid.UUID]bool{}
		}
		followups[c.EventID][c.UserID] = c.Prefs.EventFollowups
	}

	if len(followups[open]) != 2 {
		t.Fatalf("an event nobody has answered came back for %d people, want 2", len(followups[open]))
	}
	// The switch that gates "how was it?" is the one read, not event_reminders.
	if !followups[open][a] || followups[open][b] {
		t.Errorf("event_followups = %v, want Ada on and Bo off", followups[open])
	}
	if len(followups[done]) != 0 || len(followups[didnt]) != 0 {
		t.Errorf("an event that has been answered came back: done=%v didnt_happen=%v", followups[done], followups[didnt])
	}
}
