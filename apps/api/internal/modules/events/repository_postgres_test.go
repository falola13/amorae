package events_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/events"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
	"github.com/falola13/amorae/apps/api/migrations"
)

// couple inserts a user and a couple, bare SQL like the other modules'
// repository tests: this only needs rows to exist.
func couple(t *testing.T, db *database.DB) (coupleID, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	userID, coupleID = uuid.New(), uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
		VALUES ($1, $2, 'Ada', 'hash', $3, $3)
	`, userID, "test+"+uuid.NewString()+"@example.com", now); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & Bo', 'UTC', $2, $3, $3)
	`, coupleID, userID, now); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	return coupleID, userID
}

func TestPostgresRepository_RemindersAndOutcome(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := events.NewPostgresRepository(db)
	coupleID, userID := couple(t, db)
	at := time.Now().UTC().Truncate(time.Microsecond)

	e := events.Event{
		CoupleID: coupleID, CreatedBy: &userID, Kind: events.KindTogether,
		Title: "Dinner", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		Reminders: []string{"1 day before", "at 16:00", "at the time"},
	}
	id, err := repo.Create(ctx, e, at)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.ByID(ctx, coupleID, id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if strings.Join(got.Reminders, "|") != "1 day before|at 16:00|at the time" {
		t.Errorf("reminders = %v, want them in the order given", got.Reminders)
	}
	if got.Done || got.DidntHappen {
		t.Errorf("done = %v, didnt_happen = %v, a new event has said nothing", got.Done, got.DidntHappen)
	}

	t.Run("no reminders is an empty list, not nothing", func(t *testing.T) {
		bare := e
		bare.Reminders = nil
		bareID, err := repo.Create(ctx, bare, at)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.ByID(ctx, coupleID, bareID)
		if err != nil {
			t.Fatalf("ByID: %v", err)
		}
		if len(got.Reminders) != 0 {
			t.Errorf("reminders = %v", got.Reminders)
		}
	})

	t.Run("an update replaces the list", func(t *testing.T) {
		edit := got
		edit.Reminders = []string{"the morning of"}
		if err := repo.Update(ctx, edit, false, at); err != nil {
			t.Fatalf("Update: %v", err)
		}
		after, _ := repo.ByID(ctx, coupleID, id)
		if len(after.Reminders) != 1 || after.Reminders[0] != "the morning of" {
			t.Errorf("reminders = %v", after.Reminders)
		}
	})

	t.Run("the outcome is one answer", func(t *testing.T) {
		for _, tc := range []struct {
			done, didnt bool
		}{{true, false}, {false, true}, {false, false}} {
			if err := repo.SetOutcome(ctx, coupleID, id, tc.done, tc.didnt, at); err != nil {
				t.Fatalf("SetOutcome: %v", err)
			}
			after, _ := repo.ByID(ctx, coupleID, id)
			if after.Done != tc.done || after.DidntHappen != tc.didnt {
				t.Errorf("done = %v, didnt_happen = %v, want %v, %v", after.Done, after.DidntHappen, tc.done, tc.didnt)
			}
		}
	})

	t.Run("the database refuses both at once", func(t *testing.T) {
		if err := repo.SetOutcome(ctx, coupleID, id, true, true, at); err == nil {
			t.Error("done and didnt_happen were both stored")
		}
	})
}

// The migration is the thing being tested: it runs its own Down and then its
// own Up inside a transaction that is thrown away, so the shared test
// database is left as it was found and nothing else sees the old shape.
func TestMigration_BackfillsReminders(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, userID := couple(t, db)

	raw, err := migrations.FS.ReadFile("00034_event_reminders_and_outcome.sql")
	if err != nil {
		t.Fatalf("reading the migration: %v", err)
	}
	up, down, _ := strings.Cut(string(raw), "-- +goose Down")
	up = strings.TrimPrefix(up, "-- +goose Up")

	statements := func(script string) []string {
		var lines []string
		for _, line := range strings.Split(script, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				lines = append(lines, line)
			}
		}
		var out []string
		for _, s := range strings.Split(strings.Join(lines, "\n"), ";") {
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}

	errRollback := errors.New("roll back")
	err = db.InTx(ctx, func(ctx context.Context) error {
		for _, s := range statements(down) {
			if _, err := db.Q(ctx).Exec(ctx, s); err != nil {
				t.Fatalf("down: %v\n%s", err, s)
			}
		}
		withOne, withNone, withBlank := uuid.New(), uuid.New(), uuid.New()
		for id, reminder := range map[uuid.UUID]*string{
			withOne: ptr("1 hour before"), withNone: nil, withBlank: ptr(""),
		} {
			if _, err := db.Q(ctx).Exec(ctx, `
				INSERT INTO events (id, couple_id, title, date, reminder, created_by, kind, created_at, updated_at)
				VALUES ($1, $2, 'Dinner', '2026-10-10', $3, $4, 'together', now(), now())
			`, id, coupleID, reminder, userID); err != nil {
				t.Fatalf("seeding an old-shape event: %v", err)
			}
		}

		for _, s := range statements(up) {
			if _, err := db.Q(ctx).Exec(ctx, s); err != nil {
				t.Fatalf("up: %v\n%s", err, s)
			}
		}

		for id, want := range map[uuid.UUID]string{withOne: "1 hour before", withNone: "", withBlank: ""} {
			var got []string
			var didnt bool
			if err := db.Q(ctx).QueryRow(ctx,
				`SELECT reminders, didnt_happen FROM events WHERE id = $1`, id).Scan(&got, &didnt); err != nil {
				t.Fatalf("reading it back: %v", err)
			}
			if want == "" && len(got) != 0 || want != "" && (len(got) != 1 || got[0] != want) {
				t.Errorf("reminders = %v, want [%q]", got, want)
			}
			if didnt {
				t.Error("an old event came out as having not happened")
			}
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("InTx: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
