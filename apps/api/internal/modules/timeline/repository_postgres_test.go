package timeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/timeline"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// pair inserts two users and a live couple they both belong to, timezone
// fixed to UTC so this test's own date math needs no timezone conversion of
// its own. Deliberately bare SQL rather than the couples/user packages, the
// same reasoning milestones' repository test gives for doing this itself.
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

func sundayOf(t time.Time) time.Time {
	y, m, d := t.AddDate(0, 0, -int(t.Weekday())).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// seed writes exactly one of each of the seven sources timeline draws from,
// spaced far enough apart in time that their "at" values can't tie
// regardless of clock skew between Go's now and Postgres's now(). It
// returns them oldest first (index 6 is newest) so a test can name items by
// their expected position without re-deriving the order itself.
func seed(t *testing.T, db *database.DB, coupleID, a uuid.UUID) []timeline.Item {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	exec := func(query string, args ...any) {
		if _, err := db.Q(ctx).Exec(ctx, query, args...); err != nil {
			t.Fatalf("seeding: %v: %v", query, err)
		}
	}

	// A prayer week from three weeks ago, published, with three points —
	// oldest of everything here, since even its end is well over a week gone.
	weekStart := sundayOf(today.AddDate(0, 0, -21))
	weekID := uuid.New()
	exec(`
		INSERT INTO prayer_weeks (id, couple_id, week_start, setter_user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'published', $5, $5)
	`, weekID, coupleID, weekStart, a, now)
	for i := 0; i < 3; i++ {
		exec(`
			INSERT INTO prayer_points (id, week_id, position, title, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
		`, uuid.New(), weekID, i, "Point", now)
	}

	// The answered prayer lives in a draft week of its own, so it never adds
	// a fourth point to the published week's "N prayers" count above.
	draftWeekID := uuid.New()
	exec(`
		INSERT INTO prayer_weeks (id, couple_id, week_start, setter_user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'draft', $5, $5)
	`, draftWeekID, coupleID, sundayOf(today), a, now)
	answeredAt := now.Add(-2 * time.Hour)
	answeredPointID := uuid.New()
	exec(`
		INSERT INTO prayer_points (id, week_id, position, title, answered_at, answered_by, created_at, updated_at)
		VALUES ($1, $2, 0, 'Healing', $3, $4, $5, $5)
	`, answeredPointID, draftWeekID, answeredAt, a, now)

	// A memory from five days ago — older than the event, newer than the week.
	memoryID := uuid.New()
	memoryDate := today.AddDate(0, 0, -5)
	exec(`
		INSERT INTO memories (id, couple_id, title, date, location, created_at, updated_at)
		VALUES ($1, $2, 'A quiet Saturday', $3, 'Lagos', $4, $4)
	`, memoryID, coupleID, memoryDate, now)

	// A done event from four days ago, no location — sub should fall back to "Done".
	eventID := uuid.New()
	eventDate := today.AddDate(0, 0, -4)
	exec(`
		INSERT INTO events (id, couple_id, title, date, done, created_by, kind, created_at, updated_at)
		VALUES ($1, $2, 'Dinner out', $3, true, $4, 'together', $5, $5)
	`, eventID, coupleID, eventDate, a, now)

	// A finished goal, three hours ago.
	goalID := uuid.New()
	goalUpdatedAt := now.Add(-3 * time.Hour)
	exec(`
		INSERT INTO goals (id, couple_id, title, target, unit, start_date, end_date, done, created_at, updated_at)
		VALUES ($1, $2, 'Save for the trip', 1000, 'count', $3, $4, true, $5, $6)
	`, goalID, coupleID, today.AddDate(0, 0, -30), today, now, goalUpdatedAt)

	// A journal entry, one hour ago.
	journalID := uuid.New()
	journalCreatedAt := now.Add(-1 * time.Hour)
	exec(`
		INSERT INTO journal_entries (id, couple_id, author_id, date, tag, text, created_at)
		VALUES ($1, $2, $3, $4, 'Gratitude', 'For today.', $5)
	`, journalID, coupleID, a, today, journalCreatedAt)

	// An appreciation, right now — the newest thing here.
	appreciationID := uuid.New()
	exec(`
		INSERT INTO appreciations (id, couple_id, from_id, date, text, created_at)
		VALUES ($1, $2, $3, $4, 'Thank you for today.', $5)
	`, appreciationID, coupleID, a, today, now)

	return []timeline.Item{
		{Type: timeline.TypePrayerWeek, ID: weekID},
		{Type: timeline.TypeMemory, ID: memoryID},
		{Type: timeline.TypeEvent, ID: eventID},
		{Type: timeline.TypeGoal, ID: goalID},
		{Type: timeline.TypePrayerAnswered, ID: answeredPointID},
		{Type: timeline.TypeJournal, ID: journalID},
		{Type: timeline.TypeAppreciation, ID: appreciationID},
	}
}

func typesOf(items []timeline.Item) []timeline.Type {
	out := make([]timeline.Type, len(items))
	for i, it := range items {
		out[i] = it.Type
	}
	return out
}

func TestTimeline_OrderAndFields(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	seeded := seed(t, db, coupleID, a)
	// seed returns oldest first; the timeline itself is newest first.
	wantOrder := make([]timeline.Type, len(seeded))
	for i, it := range seeded {
		wantOrder[len(seeded)-1-i] = it.Type
	}

	got, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(timeline.FilterAll), nil, 20)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if len(got) != 7 {
		t.Fatalf("got %d items, want 7: %+v", len(got), typesOf(got))
	}
	if gotTypes := typesOf(got); !equalTypes(gotTypes, wantOrder) {
		t.Fatalf("order = %v, want %v", gotTypes, wantOrder)
	}

	for i := 1; i < len(got); i++ {
		if got[i].At.After(got[i-1].At) {
			t.Errorf("item %d (%s) is later than item %d (%s): not newest-first", i, got[i].Type, i-1, got[i-1].Type)
		}
	}

	byType := map[timeline.Type]timeline.Item{}
	for _, it := range got {
		byType[it.Type] = it
	}

	if pw := byType[timeline.TypePrayerWeek]; pw.Sub != "3 prayers" {
		t.Errorf("prayer_week sub = %q, want %q", pw.Sub, "3 prayers")
	}
	if pw := byType[timeline.TypePrayerWeek]; pw.Path == "" || pw.ActorID != uuid.Nil {
		t.Errorf("prayer_week path = %q actor = %v, want a path and no actor", pw.Path, pw.ActorID)
	}
	if pa := byType[timeline.TypePrayerAnswered]; pa.Sub != "Answered" || pa.ActorID != a || pa.Path != "/prayers/answered" {
		t.Errorf("prayer_answered = %+v", pa)
	}
	if m := byType[timeline.TypeMemory]; m.Title != "A quiet Saturday" || m.Sub != "Lagos" || m.Path != "/together/memories" {
		t.Errorf("memory = %+v", m)
	}
	if e := byType[timeline.TypeEvent]; e.Sub != "Done" || e.ActorID != a {
		t.Errorf("event = %+v, want sub Done and actor set", e)
	}
	if g := byType[timeline.TypeGoal]; g.Sub != "Reached" || g.Path != "/together/goals" {
		t.Errorf("goal = %+v", g)
	}
	if j := byType[timeline.TypeJournal]; j.Title != "Gratitude" || j.Sub != "For today." || j.ActorID != a {
		t.Errorf("journal = %+v", j)
	}
	if ap := byType[timeline.TypeAppreciation]; ap.Title != "Appreciation" || ap.Sub != "Thank you for today." || ap.ActorID != a {
		t.Errorf("appreciation = %+v", ap)
	}
}

func TestTimeline_Filters(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	seed(t, db, coupleID, a)

	cases := []struct {
		filter timeline.Filter
		want   []timeline.Type
	}{
		{timeline.FilterPrayer, []timeline.Type{timeline.TypePrayerAnswered, timeline.TypePrayerWeek}},
		{timeline.FilterMoments, []timeline.Type{timeline.TypeAppreciation, timeline.TypeJournal, timeline.TypeMemory}},
		{timeline.FilterPlans, []timeline.Type{timeline.TypeGoal, timeline.TypeEvent}},
	}
	for _, tc := range cases {
		t.Run(string(tc.filter), func(t *testing.T) {
			got, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(tc.filter), nil, 20)
			if err != nil {
				t.Fatalf("Timeline: %v", err)
			}
			if gotTypes := typesOf(got); !equalTypes(gotTypes, tc.want) {
				t.Fatalf("%s = %v, want %v", tc.filter, gotTypes, tc.want)
			}
		})
	}
}

func TestService_CursorPaging(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	seed(t, db, coupleID, a)

	svc := timeline.NewService(repo, fixedCouple{coupleID}, nil, time.Now)

	var seen []timeline.Type
	var before *time.Time
	for pages := 0; pages < 10; pages++ {
		page, err := svc.List(ctx, uuid.New(), before, timeline.FilterAll, 3)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(page.Items) == 0 {
			t.Fatal("an empty page before Next went nil")
		}
		for _, it := range page.Items {
			seen = append(seen, it.Type)
		}
		if page.Next == nil {
			break
		}
		before = page.Next
	}

	if len(seen) != 7 {
		t.Fatalf("paged through %d items, want 7: %v", len(seen), seen)
	}
	// No duplicate and no skip: every type appears, and paging never reorders
	// what one big page would have given.
	full, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(timeline.FilterAll), nil, 20)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if !equalTypes(seen, typesOf(full)) {
		t.Fatalf("paged order = %v, want %v", seen, typesOf(full))
	}
}

func TestTimeline_CoupleIsolation(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleA, aUser, _ := pair(t, db)
	seed(t, db, coupleA, aUser)

	coupleB, bUser, _ := pair(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO appreciations (id, couple_id, from_id, date, text, created_at)
		VALUES ($1, $2, $3, $4, 'Only for couple B.', $5)
	`, uuid.New(), coupleB, bUser, today, now); err != nil {
		t.Fatalf("seeding couple B: %v", err)
	}

	gotB, err := repo.Timeline(ctx, coupleB, timeline.TypesFor(timeline.FilterAll), nil, 20)
	if err != nil {
		t.Fatalf("Timeline(B): %v", err)
	}
	if len(gotB) != 1 || gotB[0].Title != "Appreciation" {
		t.Fatalf("couple B's timeline = %+v, want its own single appreciation", gotB)
	}

	gotA, err := repo.Timeline(ctx, coupleA, timeline.TypesFor(timeline.FilterAll), nil, 20)
	if err != nil {
		t.Fatalf("Timeline(A): %v", err)
	}
	for _, it := range gotA {
		if it.Sub == "Only for couple B." {
			t.Fatal("couple A's timeline included couple B's appreciation")
		}
	}
}

type fixedCouple struct{ id uuid.UUID }

func (f fixedCouple) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return f.id, nil }

func equalTypes(got, want []timeline.Type) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// A finished or left challenge is part of the story, and belongs to the
// moments filter; one still going is not.
func TestTimeline_Challenges(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	repo := timeline.NewPostgresRepository(db)

	coupleID, a, _ := pair(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	insert := func(title, status string, endedAt *time.Time, startedOn time.Time) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO challenges (id, couple_id, template, title, started_on, status, ended_at, created_by, created_at, updated_at)
			VALUES ($1, $2, 'custom', $3, $4, $5, $6, $7, $8, $8)
		`, id, coupleID, title, startedOn, status, endedAt, a, now); err != nil {
			t.Fatalf("insert challenge: %v", err)
		}
		return id
	}
	finishedAt := now.Add(-2 * time.Hour)
	endedAt := now.Add(-5 * time.Hour)
	finished := insert("Our three days", "finished", &finishedAt, today.AddDate(0, 0, -3))
	ended := insert("Advent together", "ended", &endedAt, today.AddDate(0, 0, -9))
	insert("Still going", "active", nil, today)

	got, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(timeline.FilterMoments), nil, 20)
	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if len(got) != 2 || got[0].ID != finished || got[1].ID != ended {
		t.Fatalf("moments = %+v, want the finished one then the ended one, and not the active one", got)
	}

	f, e := got[0], got[1]
	if f.Type != timeline.TypeChallenge || f.Title != "Our three days" || f.Sub != "Finished together" ||
		f.Path != "/together/challenges/"+finished.String() || !f.At.Equal(finishedAt) || f.ActorID != uuid.Nil {
		t.Errorf("finished = %+v", f)
	}
	if e.Sub != "Ended early" || e.Path != "/together/challenges/"+ended.String() || !e.At.Equal(endedAt) {
		t.Errorf("ended = %+v", e)
	}

	// Not in the other filters, but in "all".
	for _, f := range []timeline.Filter{timeline.FilterPrayer, timeline.FilterPlans} {
		other, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(f), nil, 20)
		if err != nil || len(other) != 0 {
			t.Errorf("%s = %+v, err %v; want nothing", f, other, err)
		}
	}
	all, err := repo.Timeline(ctx, coupleID, timeline.TypesFor(timeline.FilterAll), nil, 20)
	if err != nil || len(all) != 2 {
		t.Errorf("all = %+v, err %v; want the two kept challenges", all, err)
	}
}
