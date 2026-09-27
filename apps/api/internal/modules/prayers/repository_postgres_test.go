package prayers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/prayers"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// fakeCouples hands the service a fixed CoupleContext, so a service-level
// test can drive prayers.Service against the real database without also
// standing up the couples module.
type fakeCouples struct{ cc prayers.CoupleContext }

func (f fakeCouples) ForPrayers(context.Context, uuid.UUID) (prayers.CoupleContext, error) {
	return f.cc, nil
}

// fakePoker stands in for the notifications worker, which this test has no
// need to stand up.
type fakePoker struct{}

func (fakePoker) Poke() {}

// countingPoker counts pokes, so a test can check exactly one happened —
// or that none did, for the paths that shouldn't produce one.
type countingPoker struct{ pokes int }

func (p *countingPoker) Poke() { p.pokes++ }

// dbtest shares one database across packages, so anything unique per run has
// to be unique here too.
func uniqueEmail() string { return "prayers+" + uuid.NewString() + "@example.com" }

// pair sets up what every prayer week needs: a couple with two members.
func pair(t *testing.T, db *database.DB) (coupleID, ada, ben uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	users := user.NewPostgresRepository(db)
	repo := couples.NewPostgresRepository(db)

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
	code := "P" + uuid.NewString()[:5]
	if _, err := repo.Create(ctx, code, now.Add(7*24*time.Hour), c); err != nil {
		t.Fatalf("create couple: %v", err)
	}
	if err := repo.Join(ctx, second.ID, code, now); err != nil {
		t.Fatalf("join couple: %v", err)
	}
	return c.ID, first.ID, second.ID
}

func TestPostgresRepository_EnsureWeek_IsSafeToRunTwice(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, _ := pair(t, db)
	repo := prayers.NewPostgresRepository(db)

	weekStart := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC) // a Sunday
	at := time.Now().UTC().Truncate(time.Microsecond)

	first, err := repo.EnsureWeek(ctx, coupleID, weekStart, ada, at)
	if err != nil {
		t.Fatalf("first EnsureWeek: %v", err)
	}

	// Concurrent creation must return the same week to both callers.
	second, err := repo.EnsureWeek(ctx, coupleID, weekStart, ada, at)
	if err != nil {
		t.Fatalf("second EnsureWeek: %v", err)
	}
	if first != second {
		t.Errorf("two calls made two weeks: %s and %s", first, second)
	}

	var n int
	if err := db.Q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM prayer_weeks WHERE couple_id = $1`, coupleID).Scan(&n); err != nil {
		t.Fatalf("counting weeks: %v", err)
	}
	if n != 1 {
		t.Errorf("%d week rows, want 1", n)
	}
}

func TestPostgresRepository_LoadsAWholeWeek(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)

	weekStart := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	at := time.Now().UTC().Truncate(time.Microsecond)
	weekID, err := repo.EnsureWeek(ctx, coupleID, weekStart, ada, at)
	if err != nil {
		t.Fatalf("EnsureWeek: %v", err)
	}

	// Inserted directly: this test is about reading.
	points := make([]uuid.UUID, 3)
	for i := range points {
		points[i] = uuid.New()
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO prayer_points (id, week_id, position, title, body, scripture)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, points[i], weekID, i, []string{"Work", "Family", "Rest"}[i], "for the week", nil); err != nil {
			t.Fatalf("insert point %d: %v", i, err)
		}
	}
	// Ada prayed two of them, Ben one — all on the week's own Sunday.
	for _, c := range []struct {
		point uuid.UUID
		user  uuid.UUID
	}{{points[0], ada}, {points[1], ada}, {points[0], ben}} {
		if _, err := db.Q(ctx).Exec(ctx,
			`INSERT INTO prayer_completions (point_id, user_id, prayed_on) VALUES ($1, $2, $3)`,
			c.point, c.user, weekStart); err != nil {
			t.Fatalf("insert completion: %v", err)
		}
	}
	if _, err := db.Q(ctx).Exec(ctx,
		`INSERT INTO prayer_reflections (week_id, user_id, body) VALUES ($1, $2, $3)`,
		weekID, ben, "A hard week, but a good one."); err != nil {
		t.Fatalf("insert reflection: %v", err)
	}

	rec, err := repo.WeekStarting(ctx, coupleID, weekStart)
	if err != nil {
		t.Fatalf("WeekStarting: %v", err)
	}

	t.Run("the week itself", func(t *testing.T) {
		if rec.ID != weekID {
			t.Errorf("id = %s, want %s", rec.ID, weekID)
		}
		if rec.SetterUserID != ada {
			t.Error("the wrong person is down as the setter")
		}
		if rec.Status != prayers.StatusDraft {
			t.Errorf("status = %q, want %q", rec.Status, prayers.StatusDraft)
		}
	})

	t.Run("points come back in the order they were arranged", func(t *testing.T) {
		if len(rec.Points) != 3 {
			t.Fatalf("%d points, want 3", len(rec.Points))
		}
		for i, want := range []string{"Work", "Family", "Rest"} {
			if rec.Points[i].Title != want {
				t.Errorf("point %d is %q, want %q", i, rec.Points[i].Title, want)
			}
			if rec.Points[i].Position != i {
				t.Errorf("point %d has position %d", i, rec.Points[i].Position)
			}
		}
	})

	t.Run("completions are kept apart, per person", func(t *testing.T) {
		if len(rec.Completed[ada]) != 2 {
			t.Errorf("Ada completed %d, want 2", len(rec.Completed[ada]))
		}
		if len(rec.Completed[ben]) != 1 {
			t.Errorf("Ben completed %d, want 1", len(rec.Completed[ben]))
		}
	})

	t.Run("reflections are per person too", func(t *testing.T) {
		if rec.Reflections[ben] == "" {
			t.Error("Ben's reflection did not come back")
		}
		if _, wrote := rec.Reflections[ada]; wrote {
			t.Error("Ada has a reflection she never wrote")
		}
	})

}

func TestPostgresRepository_HistoryExcludesTheCurrentWeek(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)
	at := time.Now().UTC().Truncate(time.Microsecond)

	weeks := []time.Time{
		time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), // the current one
	}
	for i, ws := range weeks {
		setter := ada
		if i%2 == 1 {
			setter = ben
		}
		if _, err := repo.EnsureWeek(ctx, coupleID, ws, setter, at); err != nil {
			t.Fatalf("EnsureWeek %s: %v", ws.Format(time.DateOnly), err)
		}
	}

	history, err := repo.History(ctx, coupleID, weeks[2])
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("%d weeks in history, want 2", len(history))
	}
	// Newest first: the history screen reads backwards from now.
	if !history[0].WeekStart.Equal(weeks[1]) || !history[1].WeekStart.Equal(weeks[0]) {
		t.Errorf("history is in the wrong order: %s then %s",
			history[0].WeekStart.Format(time.DateOnly), history[1].WeekStart.Format(time.DateOnly))
	}
}

func TestPostgresRepository_ReplacePoints(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)
	at := time.Now().UTC().Truncate(time.Microsecond)
	weekStart := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)

	weekID, err := repo.EnsureWeek(ctx, coupleID, weekStart, ada, at)
	if err != nil {
		t.Fatalf("EnsureWeek: %v", err)
	}

	write := func(points []prayers.Point) prayers.Record {
		t.Helper()
		cleaned, err := prayers.ValidatePoints(points)
		if err != nil {
			t.Fatalf("ValidatePoints: %v", err)
		}
		if err := repo.ReplacePoints(ctx, weekID, cleaned, at); err != nil {
			t.Fatalf("ReplacePoints: %v", err)
		}
		rec, err := repo.WeekByID(ctx, coupleID, weekID)
		if err != nil {
			t.Fatalf("WeekByID: %v", err)
		}
		return rec
	}

	rec := write([]prayers.Point{{Title: "Work"}, {Title: "Family"}, {Title: "Rest"}})
	if len(rec.Points) != 3 {
		t.Fatalf("%d points after the first write, want 3", len(rec.Points))
	}
	work, family, rest := rec.Points[0], rec.Points[1], rec.Points[2]
	day := weekStart
	nextDay := weekStart.AddDate(0, 0, 1)

	t.Run("reordering keeps the points, and what was prayed on them", func(t *testing.T) {
		// Ben prays the first one on two different days, then Ada rearranges
		// the week.
		if err := repo.SetCompletion(ctx, work.ID, ben, day, true, at); err != nil {
			t.Fatalf("SetCompletion (day 1): %v", err)
		}
		if err := repo.SetCompletion(ctx, work.ID, ben, nextDay, true, at); err != nil {
			t.Fatalf("SetCompletion (day 2): %v", err)
		}

		// Exercises the UNIQUE (week_id, position) collision guard.
		rec := write([]prayers.Point{
			{ID: rest.ID, Title: "Rest"},
			{ID: work.ID, Title: "Work"},
			{ID: family.ID, Title: "Family"},
		})
		for i, want := range []string{"Rest", "Work", "Family"} {
			if rec.Points[i].Title != want {
				t.Errorf("position %d is %q, want %q", i, rec.Points[i].Title, want)
			}
		}
		if rec.Points[1].ID != work.ID {
			t.Error("the point was replaced rather than moved, so its id changed")
		}
		// The id survived, so Ben's prayer went with it — once, not twice,
		// even though he prayed it on two separate days.
		if len(rec.Completed[ben]) != 1 || rec.Completed[ben][0] != work.ID {
			t.Errorf("Ben's completion did not survive the reorder: %v", rec.Completed[ben])
		}
		// But each day is still there of its own accord.
		if len(rec.ByDay[day][ben]) != 1 || len(rec.ByDay[nextDay][ben]) != 1 {
			t.Errorf("the two days of Ben's completion did not both survive: %+v", rec.ByDay)
		}
	})

	t.Run("a dropped point is gone, and takes its completions with it", func(t *testing.T) {
		rec := write([]prayers.Point{{ID: rest.ID, Title: "Rest"}})
		if len(rec.Points) != 1 {
			t.Fatalf("%d points, want 1", len(rec.Points))
		}
		if len(rec.Completed[ben]) != 0 {
			t.Errorf("Ben still has a completion on a point that no longer exists: %v", rec.Completed[ben])
		}
	})

	t.Run("an id the week does not own is treated as a new point", func(t *testing.T) {
		// A client must never get to choose a primary key.
		stranger := uuid.New()
		rec := write([]prayers.Point{{ID: rest.ID, Title: "Rest"}, {ID: stranger, Title: "Smuggled"}})
		if len(rec.Points) != 2 {
			t.Fatalf("%d points, want 2", len(rec.Points))
		}
		if rec.Points[1].ID == stranger {
			t.Error("the client chose its own primary key")
		}
	})
}

func TestPostgresRepository_PublishAndComplete(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)
	at := time.Now().UTC().Truncate(time.Microsecond)
	weekStart := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)

	weekID, err := repo.EnsureWeek(ctx, coupleID, weekStart, ada, at)
	if err != nil {
		t.Fatalf("EnsureWeek: %v", err)
	}
	// Rest is every day; Work is Sunday and Monday only, so the days
	// breakdown below has something to distinguish.
	cleaned, err := prayers.ValidatePoints([]prayers.Point{
		{Title: "Rest"},
		{Title: "Work", WeekdaysRaw: []int{0, 1}},
	})
	if err != nil {
		t.Fatalf("ValidatePoints: %v", err)
	}
	if err := repo.ReplacePoints(ctx, weekID, cleaned, at); err != nil {
		t.Fatalf("ReplacePoints: %v", err)
	}
	day := weekStart // a Sunday: both points are scheduled on it

	t.Run("publishing twice leaves one published week, crediting the first publisher", func(t *testing.T) {
		if err := repo.Publish(ctx, weekID, ada, at); err != nil {
			t.Fatalf("first Publish: %v", err)
		}
		later := at.Add(time.Hour)
		if err := repo.Publish(ctx, weekID, ben, later); err != nil {
			t.Fatalf("second Publish: %v", err)
		}
		rec, err := repo.WeekByID(ctx, coupleID, weekID)
		if err != nil {
			t.Fatalf("WeekByID: %v", err)
		}
		if rec.Status != prayers.StatusPublished {
			t.Errorf("status = %q, want published", rec.Status)
		}
		// A retry must not move the moment it was shared, nor credit
		// whoever happened to call publish again.
		if rec.PublishedAt == nil || !rec.PublishedAt.Equal(at) {
			t.Errorf("published_at = %v, want %v", rec.PublishedAt, at)
		}
		if rec.PublishedBy == nil || *rec.PublishedBy != ada {
			t.Errorf("published_by = %v, want %s (the first publisher)", rec.PublishedBy, ada)
		}
	})

	t.Run("completing twice on the same day is the same as completing once", func(t *testing.T) {
		rec, _ := repo.WeekByID(ctx, coupleID, weekID)
		point := rec.Points[0].ID
		for i := 0; i < 2; i++ {
			if err := repo.SetCompletion(ctx, point, ben, day, true, at); err != nil {
				t.Fatalf("SetCompletion %d: %v", i, err)
			}
		}
		rec, _ = repo.WeekByID(ctx, coupleID, weekID)
		if len(rec.Completed[ben]) != 1 {
			t.Errorf("Ben has %d completions, want 1", len(rec.Completed[ben]))
		}
		if len(rec.Completed[ada]) != 0 {
			t.Error("Ada was marked as having prayed something she never touched")
		}

		if err := repo.SetCompletion(ctx, point, ben, day, false, at); err != nil {
			t.Fatalf("clearing: %v", err)
		}
		rec, _ = repo.WeekByID(ctx, coupleID, weekID)
		if len(rec.Completed[ben]) != 0 {
			t.Error("unmarking left the completion behind")
		}
	})

	t.Run("the days breakdown reflects each point's own schedule", func(t *testing.T) {
		var rest, work prayers.Point
		rec, _ := repo.WeekByID(ctx, coupleID, weekID)
		for _, p := range rec.Points {
			switch p.Title {
			case "Rest":
				rest = p
			case "Work":
				work = p
			}
		}
		if err := repo.SetCompletion(ctx, rest.ID, ada, day, true, at); err != nil {
			t.Fatalf("SetCompletion (rest): %v", err)
		}
		if err := repo.SetCompletion(ctx, work.ID, ben, day, true, at); err != nil {
			t.Fatalf("SetCompletion (work): %v", err)
		}

		rec, err := repo.WeekByID(ctx, coupleID, weekID)
		if err != nil {
			t.Fatalf("WeekByID: %v", err)
		}
		if len(rec.ByDay[day][ada]) != 1 || rec.ByDay[day][ada][0] != rest.ID {
			t.Errorf("Ada's day-1 completions = %v, want just %s", rec.ByDay[day][ada], rest.ID)
		}
		if len(rec.ByDay[day][ben]) != 1 || rec.ByDay[day][ben][0] != work.ID {
			t.Errorf("Ben's day-1 completions = %v, want just %s", rec.ByDay[day][ben], work.ID)
		}
		// Clean up so later subtests see a blank slate.
		_ = repo.SetCompletion(ctx, rest.ID, ada, day, false, at)
		_ = repo.SetCompletion(ctx, work.ID, ben, day, false, at)
	})

	t.Run("a reflection replaces rather than accumulates", func(t *testing.T) {
		if err := repo.SetReflection(ctx, weekID, ada, "A hard week.", at); err != nil {
			t.Fatalf("SetReflection: %v", err)
		}
		if err := repo.SetReflection(ctx, weekID, ada, "A better week.", at); err != nil {
			t.Fatalf("SetReflection again: %v", err)
		}
		rec, _ := repo.WeekByID(ctx, coupleID, weekID)
		if rec.Reflections[ada] != "A better week." {
			t.Errorf("reflection = %q", rec.Reflections[ada])
		}
		if err := repo.SetReflection(ctx, weekID, ada, "", at); err != nil {
			t.Fatalf("clearing reflection: %v", err)
		}
		rec, _ = repo.WeekByID(ctx, coupleID, weekID)
		if _, still := rec.Reflections[ada]; still {
			t.Error("an emptied reflection is still there")
		}
	})
}

func TestService_SetCompletion_OnlyTodaysPointsInTheCurrentWeek(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)

	cc := prayers.CoupleContext{
		CoupleID: coupleID,
		Location: time.UTC,
		Members: []prayers.Member{
			{UserID: ada, JoinedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{UserID: ben, JoinedAt: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)},
		},
	}
	// A fixed Tuesday, so "today" and "this week" are both under the test's
	// control rather than the wall clock's.
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	svc := prayers.NewService(repo, fakeCouples{cc: cc}, func() time.Time { return now }, fakePoker{})

	rec, _, err := svc.SavePoints(ctx, ada, []prayers.Point{
		{Title: "Every day"},
		{Title: "Mondays only", WeekdaysRaw: []int{1}},
	})
	if err != nil {
		t.Fatalf("SavePoints: %v", err)
	}
	everyDay, mondaysOnly := rec.Points[0].ID, rec.Points[1].ID

	t.Run("an unpublished week has nothing to complete", func(t *testing.T) {
		if _, _, err := svc.SetCompletion(ctx, ada, everyDay, true); !errors.Is(err, prayers.ErrNotShared) {
			t.Errorf("SetCompletion on a draft = %v, want ErrNotShared", err)
		}
	})

	if _, _, err := svc.Publish(ctx, ben); err != nil {
		t.Fatalf("Publish (by the non-setter, DEC-33): %v", err)
	}

	t.Run("a point scheduled for today may be completed, by either partner", func(t *testing.T) {
		if _, _, err := svc.SetCompletion(ctx, ada, everyDay, true); err != nil {
			t.Errorf("SetCompletion by the setter = %v, want nil", err)
		}
		if _, _, err := svc.SetCompletion(ctx, ben, everyDay, true); err != nil {
			t.Errorf("SetCompletion by the partner = %v, want nil", err)
		}
	})

	t.Run("a point not scheduled for today is refused", func(t *testing.T) {
		// `now` is a Tuesday; this point only runs on Mondays.
		if _, _, err := svc.SetCompletion(ctx, ada, mondaysOnly, true); !errors.Is(err, prayers.ErrNotForToday) {
			t.Errorf("SetCompletion(off-schedule) = %v, want ErrNotForToday", err)
		}
	})

	t.Run("a point outside the current week is refused", func(t *testing.T) {
		// A published week from last month, with its own "every day" point.
		pastStart := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
		pastWeekID, err := repo.EnsureWeek(ctx, coupleID, pastStart, ada, now)
		if err != nil {
			t.Fatalf("EnsureWeek (past): %v", err)
		}
		cleaned, err := prayers.ValidatePoints([]prayers.Point{{Title: "Long done"}})
		if err != nil {
			t.Fatalf("ValidatePoints: %v", err)
		}
		if err := repo.ReplacePoints(ctx, pastWeekID, cleaned, now); err != nil {
			t.Fatalf("ReplacePoints (past): %v", err)
		}
		if err := repo.Publish(ctx, pastWeekID, ada, now); err != nil {
			t.Fatalf("Publish (past): %v", err)
		}
		past, err := repo.WeekByID(ctx, coupleID, pastWeekID)
		if err != nil {
			t.Fatalf("WeekByID (past): %v", err)
		}

		if _, _, err := svc.SetCompletion(ctx, ada, past.Points[0].ID, true); !errors.Is(err, prayers.ErrNotForToday) {
			t.Errorf("SetCompletion(past week) = %v, want ErrNotForToday", err)
		}
	})
}

// TestService_Pokes checks each of prayers' immediate-notification write
// paths against a real database: Publish, marking a point done, and marking
// one answered each poke exactly once; taking a mark or an answer back
// pokes nobody, since neither has a partner-facing notification of its own.
func TestService_Pokes(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, ben := pair(t, db)
	repo := prayers.NewPostgresRepository(db)

	cc := prayers.CoupleContext{
		CoupleID: coupleID,
		Location: time.UTC,
		Members: []prayers.Member{
			{UserID: ada, JoinedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{UserID: ben, JoinedAt: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)},
		},
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) // a Tuesday
	poker := &countingPoker{}
	svc := prayers.NewService(repo, fakeCouples{cc: cc}, func() time.Time { return now }, poker)

	rec, _, err := svc.SavePoints(ctx, ada, []prayers.Point{{Title: "Every day"}})
	if err != nil {
		t.Fatalf("SavePoints: %v", err)
	}
	point := rec.Points[0].ID

	t.Run("publishing pokes", func(t *testing.T) {
		poker.pokes = 0
		if _, _, err := svc.Publish(ctx, ben); err != nil {
			t.Fatalf("Publish: %v", err)
		}
		if poker.pokes != 1 {
			t.Errorf("pokes = %d, want 1", poker.pokes)
		}
	})

	t.Run("marking a point done pokes; taking it back does not", func(t *testing.T) {
		poker.pokes = 0
		if _, _, err := svc.SetCompletion(ctx, ada, point, true); err != nil {
			t.Fatalf("SetCompletion(true): %v", err)
		}
		if poker.pokes != 1 {
			t.Errorf("pokes = %d, want 1 after marking done", poker.pokes)
		}

		poker.pokes = 0
		if _, _, err := svc.SetCompletion(ctx, ada, point, false); err != nil {
			t.Fatalf("SetCompletion(false): %v", err)
		}
		if poker.pokes != 0 {
			t.Errorf("pokes = %d, want 0 after unmarking, which nobody is told about", poker.pokes)
		}
	})

	t.Run("marking a prayer answered pokes; taking that back does not", func(t *testing.T) {
		poker.pokes = 0
		if _, _, err := svc.SetAnswered(ctx, ada, point, true, ""); err != nil {
			t.Fatalf("SetAnswered(true): %v", err)
		}
		if poker.pokes != 1 {
			t.Errorf("pokes = %d, want 1 after marking answered", poker.pokes)
		}

		poker.pokes = 0
		if _, _, err := svc.SetAnswered(ctx, ada, point, false, ""); err != nil {
			t.Fatalf("SetAnswered(false): %v", err)
		}
		if poker.pokes != 0 {
			t.Errorf("pokes = %d, want 0 after taking the mark back", poker.pokes)
		}
	})
}
