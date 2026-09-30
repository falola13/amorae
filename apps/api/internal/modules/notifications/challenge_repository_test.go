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

// seedChallenge writes a three-day challenge that began today in the couple's
// zone (UTC here), with the given status, and returns its day ids in order.
func seedChallenge(t *testing.T, db *database.DB, coupleID uuid.UUID, status string) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	id := uuid.New()
	var endedAt *time.Time
	if status != "active" {
		endedAt = &now
	}
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO challenges (id, couple_id, template, title, started_on, status, ended_at, created_at, updated_at)
		VALUES ($1, $2, 'custom', 'Three days', ($3 AT TIME ZONE 'UTC')::date, $4, $5, $3, $3)
	`, id, coupleID, now, status, endedAt); err != nil {
		t.Fatalf("insert challenge: %v", err)
	}
	var days []uuid.UUID
	for n := 1; n <= 3; n++ {
		day := uuid.New()
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO challenge_days (id, challenge_id, n, prompt) VALUES ($1, $2, $3, 'Do it.')
		`, day, id, n); err != nil {
			t.Fatalf("insert day: %v", err)
		}
		days = append(days, day)
	}
	return id, days
}

func progress(t *testing.T, db *database.DB, day, user uuid.UUID, mark *string, note string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO challenge_progress (day_id, user_id, mark, marked_at, note)
		VALUES ($1, $2, $3::challenge_mark, now(), $4)
	`, day, user, mark, note); err != nil {
		t.Fatalf("insert progress: %v", err)
	}
}

func strp(s string) *string { return &s }

func candidatesFor(t *testing.T, repo *notifications.PostgresRepository, challenge uuid.UUID) map[uuid.UUID]notifications.ChallengeCandidate {
	t.Helper()
	all, err := repo.LiveChallenges(context.Background())
	if err != nil {
		t.Fatalf("LiveChallenges: %v", err)
	}
	out := map[uuid.UUID]notifications.ChallengeCandidate{}
	for _, c := range all {
		if c.ChallengeID == challenge {
			out[c.UserID] = c
		}
	}
	return out
}

// The reminder is about today's open day, for an active challenge only, and
// stops once that person has marked it — a note on its own doesn't settle it.
func TestPostgresRepository_LiveChallenges_TodaysOpenUnmarkedDay(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)
	id, days := seedChallenge(t, db, coupleID, "active")

	got := candidatesFor(t, repo, id)
	if len(got) != 2 {
		t.Fatalf("candidates = %d, want both partners", len(got))
	}
	for who, c := range got {
		if c.Day != 1 || c.Days != 3 || c.MarkedToday {
			t.Errorf("%v: day %d of %d, marked = %v; want day 1 of 3, unmarked", who, c.Day, c.Days, c.MarkedToday)
		}
	}

	// A note without a mark leaves it waiting.
	progress(t, db, days[0], a, nil, "Thinking.")
	if c := candidatesFor(t, repo, id)[a]; c.MarkedToday {
		t.Error("a note on its own settled today's day")
	}

	// Ada marks it; Bo, who hasn't, is still to be reminded.
	if _, err := db.Q(context.Background()).Exec(context.Background(), `
		UPDATE challenge_progress SET mark = 'done' WHERE day_id = $1 AND user_id = $2
	`, days[0], a); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got = candidatesFor(t, repo, id)
	if !got[a].MarkedToday || got[b].MarkedToday {
		t.Errorf("Ada marked = %v, Bo marked = %v; want true, false", got[a].MarkedToday, got[b].MarkedToday)
	}

	// A marked day 2 is not today's day; only today's counts.
	progress(t, db, days[1], b, strp("done"), "")
	if candidatesFor(t, repo, id)[b].MarkedToday {
		t.Error("a mark on tomorrow's day settled today's")
	}
}

func TestPostgresRepository_LiveChallenges_NotOnceItIsOver(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	coupleID, _, _ := pair(t, db)
	finished, _ := seedChallenge(t, db, coupleID, "finished")

	if got := candidatesFor(t, repo, finished); len(got) != 0 {
		t.Errorf("a finished challenge has %d reminders", len(got))
	}

	coupleID2, _, _ := pair(t, db)
	ended, _ := seedChallenge(t, db, coupleID2, "ended")
	if got := candidatesFor(t, repo, ended); len(got) != 0 {
		t.Errorf("a left challenge has %d reminders", len(got))
	}
}

// "Both of you" still works with the new columns: it fires for the active one
// and for the one the last mark just finished, and not for one that was left.
func TestPostgresRepository_BothMarkedDays_ActiveAndJustFinishedOnly(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	since := time.Now().Add(-time.Hour)

	forChallenge := func(id uuid.UUID) int {
		t.Helper()
		all, err := repo.BothMarkedDays(context.Background(), since)
		if err != nil {
			t.Fatalf("BothMarkedDays: %v", err)
		}
		n := 0
		for _, c := range all {
			if c.ChallengeID == id {
				n++
			}
		}
		return n
	}

	for status, want := range map[string]int{"active": 2, "finished": 2, "ended": 0} {
		coupleID, a, b := pair(t, db)
		id, days := seedChallenge(t, db, coupleID, status)
		progress(t, db, days[0], a, strp("done"), "")
		progress(t, db, days[0], b, strp("done"), "")
		if got := forChallenge(id); got != want {
			t.Errorf("%s: %d notifications, want %d", status, got, want)
		}
	}

	// One partner having only written a note is not "both".
	coupleID, a, b := pair(t, db)
	id, days := seedChallenge(t, db, coupleID, "active")
	progress(t, db, days[0], a, strp("done"), "")
	progress(t, db, days[0], b, nil, "Just a thought.")
	if got := forChallenge(id); got != 0 {
		t.Errorf("a note counted as a mark: %d notifications", got)
	}
}
