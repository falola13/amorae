package challenges_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/challenges"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/database"
	"github.com/falola13/amorae/apps/api/internal/platform/database/dbtest"
)

// pair inserts two users and a live couple they both belong to, in the given
// timezone. Bare SQL rather than the couples package, for the same reason
// timeline's repository test gives.
func pair(t *testing.T, db *database.DB, zone string) (coupleID, a, b uuid.UUID) {
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
		INSERT INTO couples (id, name, timezone, created_by, created_at, updated_at)
		VALUES ($1, 'Ada & Bo', $2, $3, $4, $4)
	`, coupleID, zone, a, now); err != nil {
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

type fixedCouple struct{ id uuid.UUID }

func (f fixedCouple) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return f.id, nil }

type noPoke struct{}

func (noPoke) Poke() {}

// clock is a settable "now" for a service under test.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newService(t *testing.T, zone string, start time.Time) (*challenges.Service, *clock, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := dbtest.New(t)
	coupleID, a, b := pair(t, db, zone)
	clk := &clock{t: start}
	svc := challenges.NewService(challenges.NewPostgresRepository(db), fixedCouple{coupleID}, clk.now, noPoke{})
	return svc, clk, coupleID, a, b
}

var (
	yes = true
	no  = false
)

func mustMark(t *testing.T, svc *challenges.Service, who uuid.UUID, n int, done *bool, skipped *bool, note *string) challenges.Viewer {
	t.Helper()
	v, err := svc.Mark(context.Background(), who, n, done, skipped, note)
	if err != nil {
		t.Fatalf("Mark day %d: %v", n, err)
	}
	return v
}

func code(t *testing.T, err error) string {
	t.Helper()
	ae, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want an apperr", err)
	}
	return ae.Code
}

// custom is three days, the shortest a couple can write.
func custom() challenges.StartInput {
	return challenges.StartInput{Custom: &challenges.Custom{
		Title: "Our three days", Prompts: []string{"One.", "Two.", "Three."},
	}}
}

func TestStart_RecordsWhoStartedItAndOnlyOneIsActive(t *testing.T) {
	ctx := context.Background()
	noon := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	svc, _, _, a, _ := newService(t, "UTC", noon)

	v, err := svc.Start(ctx, a, challenges.StartInput{Template: "seven-days-of-noticing"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if v.Status != challenges.StatusActive || v.CreatedBy != a || v.Template != "seven-days-of-noticing" || len(v.Days) != 7 {
		t.Errorf("v = %+v", v.Challenge)
	}

	// A second one, of either kind, while one is going.
	if _, err := svc.Start(ctx, a, custom()); code(t, err) != "challenge_already_running" {
		t.Errorf("a second start = %v, want challenge_already_running", err)
	}
	if _, err := svc.Start(ctx, a, challenges.StartInput{Template: "ten-conversations"}); code(t, err) != "challenge_already_running" {
		t.Errorf("a second start = %v, want challenge_already_running", err)
	}
}

func TestCustom_IsStoredAsCustom(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	v, err := svc.Start(ctx, a, custom())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if v.Template != "custom" || v.Title != "Our three days" || len(v.Days) != 3 || v.Days[2].Prompt != "Three." || v.CreatedBy != a {
		t.Errorf("v = %+v", v.Challenge)
	}
}

func TestPacing_DaysOpenOnTheCouplesCalendar(t *testing.T) {
	ctx := context.Background()
	// 23:30 UTC on the 10th is already the 11th in Kiritimati
	// (UTC+14), where the couple's calendar is a day ahead.
	start := time.Date(2026, 9, 10, 23, 30, 0, 0, time.UTC)
	svc, clk, _, a, _ := newService(t, "Pacific/Kiritimati", start)

	v, err := svc.Start(ctx, a, challenges.StartInput{Template: "seven-days-of-noticing"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Started on the couple's 11th, not the UTC 10th.
	if got := v.StartedOn.Format(time.DateOnly); got != "2026-09-11" {
		t.Fatalf("started_on = %s, want 2026-09-11", got)
	}
	if v.TodayN() != 1 || !v.Opened(1) || v.Opened(2) {
		t.Fatalf("today_n = %d, day 1 open = %v, day 2 open = %v", v.TodayN(), v.Opened(1), v.Opened(2))
	}
	if got := v.DateOf(3).Format(time.DateOnly); got != "2026-09-13" {
		t.Errorf("day 3 opens on %s, want 2026-09-13", got)
	}

	// Day 2 is not here yet, for a mark or a note alone.
	if _, err := svc.Mark(ctx, a, 2, &yes, nil, nil); code(t, err) != "day_not_open" {
		t.Errorf("marking day 2 early = %v, want day_not_open", err)
	}
	note := "Early."
	if _, err := svc.Mark(ctx, a, 2, nil, nil, &note); code(t, err) != "day_not_open" {
		t.Errorf("a note on day 2 early = %v, want day_not_open", err)
	}

	// Their midnight is 10:00 UTC (UTC+14): 09:59 UTC is still the 11th there.
	clk.t = time.Date(2026, 9, 11, 9, 59, 0, 0, time.UTC)
	if _, err := svc.Mark(ctx, a, 2, &yes, nil, nil); code(t, err) != "day_not_open" {
		t.Errorf("a minute before their midnight = %v, want day_not_open", err)
	}
	clk.t = time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	v = mustMark(t, svc, a, 2, &yes, nil, nil)
	if v.TodayN() != 2 || v.Days[1].Marks[a] != challenges.MarkDone {
		t.Errorf("today_n = %d, day 2 = %+v", v.TodayN(), v.Days[1])
	}

	// A few days on: catching up on day 1 is still fine, and today_n is clamped.
	clk.t = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	v = mustMark(t, svc, a, 1, &yes, nil, nil)
	if v.TodayN() != 7 {
		t.Errorf("today_n = %d, want it held at 7", v.TodayN())
	}
}

func TestFinishing_OnTheLastMarkByBoth(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	if _, err := svc.Start(ctx, a, custom()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	clk.advance(48 * time.Hour) // all three days are open

	for n := 1; n <= 3; n++ {
		v := mustMark(t, svc, a, n, &yes, nil, nil)
		if v.Status != challenges.StatusActive {
			t.Fatalf("finished after only Ada marked day %d", n)
		}
	}
	mustMark(t, svc, b, 1, nil, &yes, nil) // skipped counts as marked
	mustMark(t, svc, b, 2, &yes, nil, nil)
	if v, err := svc.Current(ctx, a); err != nil || v.Status != challenges.StatusActive {
		t.Fatalf("Current after Bo's second day = %+v, %v; want still active", v.Status, err)
	}

	clk.advance(time.Minute)
	v := mustMark(t, svc, b, 3, &yes, nil, nil)
	if v.Status != challenges.StatusFinished || v.EndedAt == nil || !v.EndedAt.Equal(clk.t) {
		t.Fatalf("status = %q, ended_at = %v; want finished at %v", v.Status, v.EndedAt, clk.t)
	}

	// It is no longer current, and can no longer be marked.
	if _, err := svc.Current(ctx, a); !errors.Is(err, challenges.ErrNotFound) {
		t.Errorf("Current after finishing = %v, want ErrNotFound", err)
	}
	if _, err := svc.Mark(ctx, a, 1, &no, nil, nil); code(t, err) != "challenge_over" {
		t.Errorf("marking a finished challenge = %v, want challenge_over", err)
	}

	// Starting another is now allowed.
	if _, err := svc.Start(ctx, a, challenges.StartInput{Template: "ten-conversations"}); err != nil {
		t.Errorf("starting after finishing: %v", err)
	}
}

func TestFinishing_NeedsEveryDayFromEveryone(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	if _, err := svc.Start(ctx, a, custom()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	clk.advance(72 * time.Hour)

	// A note alone is not a mark; un-marking takes a day back out.
	note := "Thinking about it."
	mustMark(t, svc, b, 1, nil, nil, &note)
	mustMark(t, svc, b, 2, &yes, nil, nil)
	mustMark(t, svc, b, 3, &yes, nil, nil)
	mustMark(t, svc, a, 1, &yes, nil, nil)
	mustMark(t, svc, a, 2, &yes, nil, nil)
	mustMark(t, svc, a, 3, &yes, nil, nil)
	if v, _ := svc.Current(ctx, a); v.Status != challenges.StatusActive {
		t.Fatalf("finished with Bo's day 1 only a note")
	}
	v := mustMark(t, svc, b, 1, &yes, nil, nil)
	if v.Status != challenges.StatusFinished {
		t.Errorf("status = %q, want finished once Bo marked day 1 too", v.Status)
	}
}

func TestLeaving_KeepsItAsEnded(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	v, err := svc.Start(ctx, a, challenges.StartInput{Template: "seven-days-of-gratitude"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	note := "Grateful for the rain."
	mustMark(t, svc, a, 1, &yes, nil, &note)

	clk.advance(time.Hour)
	if err := svc.Leave(ctx, b); err != nil {
		t.Fatalf("Leave: %v", err)
	}
	if err := svc.Leave(ctx, b); !errors.Is(err, challenges.ErrNotFound) {
		t.Errorf("leaving with nothing going = %v, want ErrNotFound", err)
	}
	if _, err := svc.Current(ctx, a); !errors.Is(err, challenges.ErrNotFound) {
		t.Errorf("Current after leaving = %v, want ErrNotFound", err)
	}
	if _, err := svc.Mark(ctx, a, 1, &yes, nil, nil); code(t, err) != "challenge_over" {
		t.Errorf("marking a left challenge = %v, want challenge_over", err)
	}

	// Kept: days, marks and notes all still there.
	kept, err := svc.Get(ctx, b, v.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if kept.Status != challenges.StatusEnded || kept.EndedAt == nil || len(kept.Days) != 7 {
		t.Fatalf("kept = %+v", kept.Challenge)
	}
	if kept.Days[0].Marks[a] != challenges.MarkDone || kept.Days[0].Notes[a] != note {
		t.Errorf("day 1 = %+v, want Ada's mark and note kept", kept.Days[0])
	}

	// And a new one can begin.
	if _, err := svc.Start(ctx, b, custom()); err != nil {
		t.Errorf("starting after leaving: %v", err)
	}
}

func TestNotes_OnTheirOwnOrWithAMark(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	if _, err := svc.Start(ctx, a, custom()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// A note without a mark, on an open day.
	note := "  We talked on the balcony.  "
	v := mustMark(t, svc, a, 1, nil, nil, &note)
	if _, marked := v.Days[0].Marks[a]; marked {
		t.Error("a note alone made a mark")
	}
	if got := v.Days[0].Notes[a]; got != "We talked on the balcony." {
		t.Errorf("note = %q, want it trimmed", got)
	}

	// Then a mark leaves the note alone; the partner sees both; theirs is theirs.
	v = mustMark(t, svc, a, 1, &yes, nil, nil)
	if v.Days[0].Marks[a] != challenges.MarkDone || v.Days[0].Notes[a] != "We talked on the balcony." {
		t.Errorf("day 1 = %+v, want the mark and the note both", v.Days[0])
	}
	other := "Me too."
	v = mustMark(t, svc, b, 1, nil, &yes, &other)
	if v.Days[0].Notes[a] == "" || v.Days[0].Notes[b] != "Me too." || v.Days[0].Marks[b] != challenges.MarkSkipped {
		t.Errorf("day 1 = %+v", v.Days[0])
	}

	// Un-marking keeps the note; clearing the note as well removes the row.
	v = mustMark(t, svc, a, 1, &no, nil, nil)
	if _, marked := v.Days[0].Marks[a]; marked || v.Days[0].Notes[a] == "" {
		t.Errorf("day 1 = %+v, want the mark gone and the note kept", v.Days[0])
	}
	empty := ""
	v = mustMark(t, svc, a, 1, nil, nil, &empty)
	if v.Days[0].Notes[a] != "" {
		t.Errorf("note = %q, want it cleared", v.Days[0].Notes[a])
	}

	// Too long is a field error.
	long := make([]rune, 281)
	for i := range long {
		long[i] = 'x'
	}
	tooLong := string(long)
	_, err := svc.Mark(ctx, a, 1, nil, nil, &tooLong)
	if ae, _ := apperr.As(err); ae == nil || ae.Fields["note"] == "" {
		t.Errorf("err = %v, want a field error on note", err)
	}
}

func TestReflections_OnlyAfterItIsOver(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	v, err := svc.Start(ctx, a, custom())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if _, err := svc.Reflect(ctx, a, v.ID, "Too soon."); code(t, err) != "challenge_not_over" {
		t.Errorf("reflecting on a going challenge = %v, want challenge_not_over", err)
	}
	if _, err := svc.Reflect(ctx, a, uuid.New(), "Nothing."); code(t, err) != "challenge_not_found" {
		t.Errorf("reflecting on a stranger's = %v, want challenge_not_found", err)
	}

	clk.advance(time.Hour)
	if err := svc.Leave(ctx, a); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	got, err := svc.Reflect(ctx, a, v.ID, "  It was good to slow down.  ")
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if got.Reflections[a] != "It was good to slow down." {
		t.Errorf("reflections = %v", got.Reflections)
	}
	if _, err := svc.Reflect(ctx, b, v.ID, "Same."); err != nil {
		t.Fatalf("Reflect (Bo): %v", err)
	}
	got, _ = svc.Get(ctx, a, v.ID)
	if got.Reflections[a] == "" || got.Reflections[b] != "Same." {
		t.Errorf("reflections = %v, want both", got.Reflections)
	}

	// Editing replaces; empty removes only the caller's own.
	got, _ = svc.Reflect(ctx, a, v.ID, "Different now.")
	if got.Reflections[a] != "Different now." {
		t.Errorf("reflection = %q", got.Reflections[a])
	}
	got, _ = svc.Reflect(ctx, a, v.ID, "")
	if _, has := got.Reflections[a]; has || got.Reflections[b] != "Same." {
		t.Errorf("reflections = %v, want only Bo's left", got.Reflections)
	}
}

func TestPast_NewestFirstWithBothCounts(t *testing.T) {
	ctx := context.Background()
	svc, clk, _, a, b := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	// First: left early with one day done by Ada.
	first, err := svc.Start(ctx, a, challenges.StartInput{Template: "seven-days-of-noticing"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	mustMark(t, svc, a, 1, &yes, nil, nil)
	clk.advance(time.Hour)
	if err := svc.Leave(ctx, a); err != nil {
		t.Fatalf("Leave: %v", err)
	}

	// Second: finished by both, Bo skipping one.
	clk.advance(time.Hour)
	second, err := svc.Start(ctx, b, custom())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	clk.advance(72 * time.Hour)
	for n := 1; n <= 3; n++ {
		mustMark(t, svc, a, n, &yes, nil, nil)
	}
	mustMark(t, svc, b, 1, &yes, nil, nil)
	mustMark(t, svc, b, 2, nil, &yes, nil)
	mustMark(t, svc, b, 3, &yes, nil, nil)

	// A third, still going, is not in the list.
	if _, err := svc.Start(ctx, a, challenges.StartInput{Template: "ten-conversations"}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	past, err := svc.Past(ctx, a)
	if err != nil {
		t.Fatalf("Past: %v", err)
	}
	if len(past) != 2 {
		t.Fatalf("past = %+v, want two", past)
	}
	if past[0].ID != second.ID || past[0].Status != challenges.StatusFinished || past[0].Days != 3 ||
		past[0].MyDone != 3 || past[0].PartnerDone != 2 || past[0].Template != "custom" {
		t.Errorf("newest = %+v", past[0])
	}
	if past[1].ID != first.ID || past[1].Status != challenges.StatusEnded || past[1].Days != 7 ||
		past[1].MyDone != 1 || past[1].PartnerDone != 0 {
		t.Errorf("oldest = %+v", past[1])
	}

	// From the other side the counts swap.
	theirs, _ := svc.Past(ctx, b)
	if theirs[0].MyDone != 2 || theirs[0].PartnerDone != 3 {
		t.Errorf("Bo's view = %+v", theirs[0])
	}
}

func TestGet_NotSomeoneElsesChallenge(t *testing.T) {
	ctx := context.Background()
	svc, _, _, a, _ := newService(t, "UTC", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	if _, err := svc.Get(ctx, a, uuid.New()); !errors.Is(err, challenges.ErrNoSuchChallenge) {
		t.Errorf("Get of a stranger's = %v, want ErrNoSuchChallenge", err)
	}
}
