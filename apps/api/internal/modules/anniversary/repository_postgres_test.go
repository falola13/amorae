package anniversary_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/anniversary"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

type fixedCouple struct{ coupleID uuid.UUID }

func (f fixedCouple) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.coupleID, nil
}

func pair(t *testing.T, db *database.DB, started string) (coupleID, a, b uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a, b = uuid.New(), uuid.New()
	for id, name := range map[uuid.UUID]string{a: "Ada", b: "Bo"} {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, 'hash', $4, $4)
		`, id, "test+"+uuid.NewString()+"@example.com", name, now); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}
	coupleID = uuid.New()
	if _, err := db.Q(ctx).Exec(ctx, `
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at, relationship_start_date)
		VALUES ($1, 'Ada & Bo', 'Africa/Lagos', $2, $3, $3, $4)
	`, coupleID, a, now, started); err != nil {
		t.Fatalf("insert couple: %v", err)
	}
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.Q(ctx).Exec(ctx, `
			INSERT INTO couple_members (id, couple_id, user_id, joined_at)
			VALUES ($1, $2, $3, $4)
		`, uuid.New(), coupleID, id, now); err != nil {
			t.Fatalf("insert member: %v", err)
		}
	}
	return coupleID, a, b
}

func TestService_LettersAreSealedUntilTheDay(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, bo := pair(t, db, "2023-10-25")

	// Noon in Lagos the day before, then the morning of.
	clock := time.Date(2026, 10, 24, 11, 0, 0, 0, time.UTC)
	svc := anniversary.NewService(anniversary.NewPostgresRepository(db), fixedCouple{coupleID}, func() time.Time { return clock }, nil)

	if _, err := svc.Write(ctx, ada, "Three years, and still you."); err != nil {
		t.Fatalf("Ada writing: %v", err)
	}
	// Writing again replaces it rather than adding a second.
	if _, err := svc.Write(ctx, ada, "Three years, and still you. Always you."); err != nil {
		t.Fatalf("Ada rewriting: %v", err)
	}

	v, err := svc.Get(ctx, bo)
	if err != nil {
		t.Fatalf("Bo reading: %v", err)
	}
	if v.Theirs != nil {
		t.Fatalf("Bo read Ada's letter a day early: %q", v.Theirs.Text)
	}
	if !v.TheirsSealed || v.Mine != nil {
		t.Errorf("sealed = %v, mine = %+v; want Ada's waiting and nothing of Bo's", v.TheirsSealed, v.Mine)
	}
	if v.Occasion.Years != 3 || v.Occasion.DaysUntil != 1 {
		t.Errorf("occasion = %+v", v.Occasion)
	}

	mine, err := svc.Get(ctx, ada)
	if err != nil {
		t.Fatalf("Ada reading: %v", err)
	}
	if mine.Mine == nil || mine.Mine.Text != "Three years, and still you. Always you." {
		t.Errorf("Ada's own = %+v", mine.Mine)
	}

	// 23:30 UTC is already the 25th in Lagos: it opens on their calendar, not UTC's.
	clock = time.Date(2026, 10, 24, 23, 30, 0, 0, time.UTC)
	v, err = svc.Get(ctx, bo)
	if err != nil {
		t.Fatalf("Bo on the day: %v", err)
	}
	if v.Theirs == nil || v.Theirs.Text != "Three years, and still you. Always you." {
		t.Fatalf("on the day Bo got %+v", v.Theirs)
	}

	// The day after, it is last year's — still there, open to both.
	clock = time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	v, err = svc.Get(ctx, bo)
	if err != nil {
		t.Fatalf("Bo the day after: %v", err)
	}
	if v.Theirs != nil || len(v.Past) != 1 || v.Occasion.Years != 4 {
		t.Errorf("the day after: theirs %+v, past %d, years %d", v.Theirs, len(v.Past), v.Occasion.Years)
	}
	if _, err := svc.Write(ctx, bo, "Too soon"); err != anniversary.ErrTooEarly {
		t.Errorf("writing a year ahead: err = %v, want too early", err)
	}
}

func TestPostgresRepository_Year(t *testing.T) {
	db := dbtest.New(t)
	ctx := context.Background()
	coupleID, ada, _ := pair(t, db, "2023-10-25")
	repo := anniversary.NewPostgresRepository(db)

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := db.Q(ctx).Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	// Two memories in the year, one before it.
	for _, d := range []string{"2026-01-10", "2026-06-01", "2025-09-01"} {
		exec(`INSERT INTO memories (id, couple_id, title, date, created_at, updated_at)
		      VALUES ($1, $2, 'A day', $3, now(), now())`, uuid.New(), coupleID, d)
	}
	// A note in the year.
	exec(`INSERT INTO appreciations (id, couple_id, from_id, date, text) VALUES ($1, $2, $3, '2026-03-03', 'Thank you')`,
		uuid.New(), coupleID, ada)

	y, err := repo.Year(ctx, coupleID, time.Date(2025, 10, 25, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Year: %v", err)
	}
	if y.Memories != 2 || y.Notes != 1 {
		t.Errorf("memories %d notes %d, want 2 and 1", y.Memories, y.Notes)
	}
}
