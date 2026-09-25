package prayers_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/prayers"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

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
	// Ada prayed two of them, Ben one.
	for _, c := range []struct {
		point uuid.UUID
		user  uuid.UUID
	}{{points[0], ada}, {points[1], ada}, {points[0], ben}} {
		if _, err := db.Q(ctx).Exec(ctx,
			`INSERT INTO prayer_completions (point_id, user_id) VALUES ($1, $2)`,
			c.point, c.user); err != nil {
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

	t.Run("it answers the question CanEditPoints asks", func(t *testing.T) {
		prayed := rec.PrayedByOthers(ada)
		if !prayed[rec.Points[0].ID] {
			t.Error("the setter was told nobody had started, but Ben had")
		}

		rewritten := append([]prayers.Point(nil), rec.Points...)
		rewritten[0].Title = "Something else entirely"
		if err := prayers.CanEditPoints(rec.Week, ada, rewritten, prayed); err == nil {
			t.Error("rewriting a prayed point should be refused")
		}

		// Adding is always allowed, however far into the week it is.
		added := append(append([]prayers.Point(nil), rec.Points...), prayers.Point{Title: "One more"})
		if err := prayers.CanEditPoints(rec.Week, ada, added, prayed); err != nil {
			t.Errorf("adding a point was refused: %v", err)
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

	t.Run("reordering keeps the points, and what was prayed on them", func(t *testing.T) {
		// Ben prays the first one, then Ada rearranges the week.
		if err := repo.SetCompletion(ctx, work.ID, ben, true, at); err != nil {
			t.Fatalf("SetCompletion: %v", err)
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
		// The id survived, so Ben's prayer went with it.
		if len(rec.Completed[ben]) != 1 || rec.Completed[ben][0] != work.ID {
			t.Errorf("Ben's completion did not survive the reorder: %v", rec.Completed[ben])
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
	cleaned, err := prayers.ValidatePoints([]prayers.Point{{Title: "Work"}})
	if err != nil {
		t.Fatalf("ValidatePoints: %v", err)
	}
	if err := repo.ReplacePoints(ctx, weekID, cleaned, at); err != nil {
		t.Fatalf("ReplacePoints: %v", err)
	}

	t.Run("publishing twice leaves one published week", func(t *testing.T) {
		if err := repo.Publish(ctx, weekID, at); err != nil {
			t.Fatalf("first Publish: %v", err)
		}
		later := at.Add(time.Hour)
		if err := repo.Publish(ctx, weekID, later); err != nil {
			t.Fatalf("second Publish: %v", err)
		}
		rec, err := repo.WeekByID(ctx, coupleID, weekID)
		if err != nil {
			t.Fatalf("WeekByID: %v", err)
		}
		if rec.Status != prayers.StatusPublished {
			t.Errorf("status = %q, want published", rec.Status)
		}
		// A retry must not move the moment it was shared.
		if rec.PublishedAt == nil || !rec.PublishedAt.Equal(at) {
			t.Errorf("published_at = %v, want %v", rec.PublishedAt, at)
		}
	})

	t.Run("completing twice is the same as completing once", func(t *testing.T) {
		rec, _ := repo.WeekByID(ctx, coupleID, weekID)
		point := rec.Points[0].ID
		for i := 0; i < 2; i++ {
			if err := repo.SetCompletion(ctx, point, ben, true, at); err != nil {
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

		if err := repo.SetCompletion(ctx, point, ben, false, at); err != nil {
			t.Fatalf("clearing: %v", err)
		}
		rec, _ = repo.WeekByID(ctx, coupleID, weekID)
		if len(rec.Completed[ben]) != 0 {
			t.Error("unmarking left the completion behind")
		}
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
