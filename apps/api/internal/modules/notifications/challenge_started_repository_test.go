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

// startedChallenge writes a three-day active challenge by `by`, with the
// given kind, starting `startsIn` days from today (UTC, the couple's zone).
func startedChallenge(t *testing.T, db *database.DB, coupleID, by uuid.UUID, kind string, startsIn int) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	id := uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO challenges (id, couple_id, template, title, kind, started_on, created_by, created_at, updated_at)
		VALUES ($1, $2, 'custom', 'Three days', $3, ($4 AT TIME ZONE 'UTC')::date + $5::int, $6, $4, $4)
	`, id, coupleID, kind, now, startsIn, by); err != nil {
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

func writtenFor(t *testing.T, repo *notifications.PostgresRepository, item uuid.UUID) []notifications.WrittenCandidate {
	t.Helper()
	all, err := repo.RecentlyWritten(context.Background(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("RecentlyWritten: %v", err)
	}
	var out []notifications.WrittenCandidate
	for _, c := range all {
		if c.ItemID == item {
			out = append(out, c)
		}
	}
	return out
}

// A started challenge reaches the pipeline for the other partner only, with
// what the message needs, and settles at once.
func TestPostgresRepository_RecentlyWritten_ChallengeStarted(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)

	shared, _ := startedChallenge(t, db, coupleID, a, "together", 3)
	got := writtenFor(t, repo, shared)
	if len(got) != 1 {
		t.Fatalf("candidates = %d, want just the partner", len(got))
	}
	c := got[0]
	want := time.Now().UTC().AddDate(0, 0, 3).Truncate(24 * time.Hour)
	if c.UserID != b || c.AuthorID != a || c.AuthorName != "Ada" || c.Kind != notifications.KindChallengeStarted ||
		c.Subject != "Three days" || c.Mine || c.Settles != 0 || c.Timezone != "UTC" || !c.StartsOn.Equal(want) {
		t.Errorf("candidate = %+v (want start %v)", c, want)
	}
	if !c.Prefs.PartnerChallenges {
		t.Error("partner_challenges should default on for somebody with no saved preferences")
	}

	// A "just me" one is still announced, marked as theirs alone.
	mine, _ := startedChallenge(t, db, coupleID, a, "mine", 0)
	got = writtenFor(t, repo, mine)
	if len(got) != 1 || got[0].UserID != b || !got[0].Mine {
		t.Errorf("a mine challenge: %+v", got)
	}

	// The partner's own switch is read.
	ctx := context.Background()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_preferences (user_id, partner_challenges) VALUES ($1, false)
	`, b); err != nil {
		t.Fatalf("save preferences: %v", err)
	}
	got = writtenFor(t, repo, shared)
	if len(got) != 1 || got[0].Prefs.PartnerChallenges {
		t.Errorf("with the switch off: %+v", got)
	}
}

// Not yet begun: no reminder. Just me: the creator's reminder alone.
func TestPostgresRepository_LiveChallenges_ScheduledAndMine(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)

	scheduled, _ := startedChallenge(t, db, coupleID, a, "together", 2)
	if got := candidatesFor(t, repo, scheduled); len(got) != 0 {
		t.Errorf("a challenge that has not begun has %d reminders", len(got))
	}

	mine, _ := startedChallenge(t, db, coupleID, a, "mine", 0)
	got := candidatesFor(t, repo, mine)
	if _, ok := got[a]; !ok || len(got) != 1 {
		t.Errorf("a just-me challenge reminds %d people (Ada among them: %v), want only Ada", len(got), ok)
	}
	if _, ok := got[b]; ok {
		t.Error("Bo was reminded about Ada's own challenge")
	}

	shared, _ := startedChallenge(t, db, coupleID, a, "together", 0)
	if got := candidatesFor(t, repo, shared); len(got) != 2 {
		t.Errorf("a shared challenge reminds %d people, want both", len(got))
	}
}

// "Both of you" is for a shared challenge only.
func TestPostgresRepository_BothMarkedDays_NotForJustMe(t *testing.T) {
	db := dbtest.New(t)
	repo := notifications.NewPostgresRepository(db)
	coupleID, a, b := pair(t, db)

	mine, mineDays := startedChallenge(t, db, coupleID, a, "mine", 0)
	shared, sharedDays := startedChallenge(t, db, coupleID, a, "together", 0)
	for _, who := range []uuid.UUID{a, b} {
		progress(t, db, mineDays[0], who, strp("done"), "")
		progress(t, db, sharedDays[0], who, strp("done"), "")
	}

	all, err := repo.BothMarkedDays(context.Background(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("BothMarkedDays: %v", err)
	}
	var sawMine, sawShared bool
	for _, c := range all {
		sawMine = sawMine || c.ChallengeID == mine
		sawShared = sawShared || c.ChallengeID == shared
	}
	if sawMine || !sawShared {
		t.Errorf("both-marked for a just-me one = %v, for the shared one = %v; want false, true", sawMine, sawShared)
	}
}
