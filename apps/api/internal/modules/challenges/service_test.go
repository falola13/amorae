package challenges

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type fakeRepo struct {
	current  Challenge
	records  int
	entries  []Entry
	started  []Template
	ended    int
	reflects []string
}

func (r *fakeRepo) Latest(context.Context, uuid.UUID, time.Time) (Challenge, error) {
	return r.current, nil
}
func (r *fakeRepo) Get(_ context.Context, _, id uuid.UUID, _ time.Time) (Challenge, error) {
	if id != r.current.ID {
		return Challenge{}, ErrNoSuchChallenge
	}
	return r.current, nil
}
func (r *fakeRepo) Past(context.Context, uuid.UUID, uuid.UUID) ([]Summary, error) { return nil, nil }
func (r *fakeRepo) Start(_ context.Context, _, _ uuid.UUID, t Template, _ time.Time) (uuid.UUID, error) {
	r.started = append(r.started, t)
	return uuid.New(), nil
}
func (r *fakeRepo) Record(_ context.Context, _, _ uuid.UUID, _ int, e Entry, _ time.Time) error {
	r.records++
	r.entries = append(r.entries, e)
	return nil
}
func (r *fakeRepo) End(context.Context, uuid.UUID, time.Time) error {
	r.ended++
	return nil
}
func (r *fakeRepo) SetReflection(_ context.Context, _, _, _ uuid.UUID, text string, _ time.Time) error {
	r.reflects = append(r.reflects, text)
	return nil
}

type fakeCouples struct{ id uuid.UUID }

func (c fakeCouples) CoupleFor(context.Context, uuid.UUID) (uuid.UUID, error) { return c.id, nil }

// fakePoker counts pokes so a test can check one happened, without either
// side knowing anything about how notifications work.
type fakePoker struct{ pokes int }

func (p *fakePoker) Poke() { p.pokes++ }

// running is a seven-day challenge started on the 1st, read on `today`.
func running(couple uuid.UUID, today time.Time) Challenge {
	started := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	c := Challenge{ID: uuid.New(), CoupleID: couple, Status: StatusActive, StartedOn: started, Today: today}
	for n := 1; n <= 7; n++ {
		c.Days = append(c.Days, Day{ID: uuid.New(), N: n, Prompt: "Do a thing."})
	}
	return c
}

func onDay(n int) time.Time { return time.Date(2026, 9, n, 0, 0, 0, 0, time.UTC) }

func codeOf(t *testing.T, err error) string {
	t.Helper()
	ae, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want an apperr", err)
	}
	return ae.Code
}

func TestMark_PokesOnlyWhenMarking(t *testing.T) {
	couple := uuid.New()
	repo := &fakeRepo{current: running(couple, onDay(1))}
	poker := &fakePoker{}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, poker)

	done := true
	t.Run("marking a day pokes", func(t *testing.T) {
		if _, err := svc.Mark(context.Background(), uuid.New(), 1, &done, nil, nil); err != nil {
			t.Fatalf("Mark: %v", err)
		}
		if poker.pokes != 1 {
			t.Errorf("pokes = %d, want 1 — the both-marked notification shouldn't wait for the next cron tick", poker.pokes)
		}
		if repo.records != 1 || !repo.entries[0].SetMark {
			t.Errorf("records = %d, entries = %+v, want one mark", repo.records, repo.entries)
		}
	})

	t.Run("clearing a mark pokes nobody", func(t *testing.T) {
		poker.pokes = 0
		notDone := false
		if _, err := svc.Mark(context.Background(), uuid.New(), 1, &notDone, nil, nil); err != nil {
			t.Fatalf("Mark (clear): %v", err)
		}
		if poker.pokes != 0 {
			t.Errorf("pokes = %d, want 0 — nobody is told about a day being unmarked", poker.pokes)
		}
		if !repo.entries[1].ClearMark {
			t.Errorf("entry = %+v, want a cleared mark", repo.entries[1])
		}
	})

	t.Run("a note on its own pokes nobody", func(t *testing.T) {
		poker.pokes = 0
		note := "That was lovely."
		if _, err := svc.Mark(context.Background(), uuid.New(), 1, nil, nil, &note); err != nil {
			t.Fatalf("Mark (note): %v", err)
		}
		e := repo.entries[2]
		if poker.pokes != 0 || e.SetMark || e.ClearMark || e.Note == nil || *e.Note != note {
			t.Errorf("pokes = %d, entry = %+v, want a note and no mark change", poker.pokes, e)
		}
	})
}

func TestMark_Pacing(t *testing.T) {
	couple := uuid.New()
	done := true
	mark := func(repo *fakeRepo, n int, note *string) error {
		svc := NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{})
		var d *bool
		if note == nil {
			d = &done
		}
		_, err := svc.Mark(context.Background(), uuid.New(), n, d, nil, note)
		return err
	}

	t.Run("a day that has not opened is not here yet", func(t *testing.T) {
		repo := &fakeRepo{current: running(couple, onDay(3))}
		err := mark(repo, 4, nil)
		if codeOf(t, err) != "day_not_open" {
			t.Fatalf("err = %v, want day_not_open", err)
		}
		if repo.records != 0 {
			t.Error("a day that has not opened was written to")
		}
	})

	t.Run("nor is a note on it", func(t *testing.T) {
		repo := &fakeRepo{current: running(couple, onDay(3))}
		note := "Early."
		if codeOf(t, mark(repo, 5, &note)) != "day_not_open" {
			t.Fatal("a note on a day not yet open was allowed")
		}
	})

	t.Run("today and every earlier day can be marked", func(t *testing.T) {
		repo := &fakeRepo{current: running(couple, onDay(3))}
		for _, n := range []int{1, 2, 3} {
			if err := mark(repo, n, nil); err != nil {
				t.Errorf("day %d: %v", n, err)
			}
		}
		if repo.records != 3 {
			t.Errorf("records = %d, want 3", repo.records)
		}
	})

	t.Run("a day the challenge does not have", func(t *testing.T) {
		repo := &fakeRepo{current: running(couple, onDay(30))}
		if codeOf(t, mark(repo, 8, nil)) != "challenge_day_not_found" {
			t.Fatal("day 8 of 7 was accepted")
		}
	})

	t.Run("one that is over cannot be marked", func(t *testing.T) {
		for _, status := range []Status{StatusFinished, StatusEnded} {
			c := running(couple, onDay(3))
			c.Status = status
			repo := &fakeRepo{current: c}
			if codeOf(t, mark(repo, 1, nil)) != "challenge_over" {
				t.Errorf("a %s challenge was marked", status)
			}
			if repo.records != 0 {
				t.Errorf("a %s challenge was written to", status)
			}
		}
	})
}

func TestTodayN(t *testing.T) {
	couple := uuid.New()
	tests := []struct {
		name  string
		today time.Time
		want  int
	}{
		{"the day it starts is day one", onDay(1), 1},
		{"the next morning is day two", onDay(2), 2},
		{"the last day", onDay(7), 7},
		{"held at the last day once the calendar runs on", onDay(20), 7},
		{"held at the first before it starts", time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC), 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := running(couple, tc.today).TodayN(); got != tc.want {
				t.Errorf("TodayN = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCurrent_OnlyWhenActive(t *testing.T) {
	couple := uuid.New()
	c := running(couple, onDay(2))
	c.Status = StatusFinished
	svc := NewService(&fakeRepo{current: c}, fakeCouples{id: couple}, time.Now, &fakePoker{})

	if _, err := svc.Current(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Current on a finished challenge = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), uuid.New(), c.ID); err != nil {
		t.Errorf("Get on a finished challenge: %v", err)
	}
}

func TestStart(t *testing.T) {
	couple := uuid.New()
	run := func(in StartInput) (*fakeRepo, error) {
		repo := &fakeRepo{current: running(couple, onDay(1))}
		svc := NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{})
		_, err := svc.Start(context.Background(), uuid.New(), in)
		return repo, err
	}

	t.Run("a template of ours", func(t *testing.T) {
		repo, err := run(StartInput{Template: "seven-days-of-noticing"})
		if err != nil || len(repo.started) != 1 || repo.started[0].Key != "seven-days-of-noticing" {
			t.Fatalf("started = %+v, err = %v", repo.started, err)
		}
	})

	t.Run("one they wrote is filed as custom", func(t *testing.T) {
		repo, err := run(StartInput{Custom: &Custom{Title: "  Our month  ", Prompts: []string{" a ", "b", "c"}}})
		if err != nil || len(repo.started) != 1 {
			t.Fatalf("started = %+v, err = %v", repo.started, err)
		}
		got := repo.started[0]
		if got.Key != CustomKey || got.Title != "Our month" || got.Prompts[0] != "a" {
			t.Errorf("started = %+v, want trimmed and filed as custom", got)
		}
	})

	t.Run("not both", func(t *testing.T) {
		_, err := run(StartInput{Template: "seven-days-of-noticing", Custom: &Custom{Title: "x", Prompts: []string{"a", "b", "c"}}})
		if codeOf(t, err) != "validation_failed" {
			t.Fatalf("err = %v, want a validation error", err)
		}
	})

	t.Run("an unknown template", func(t *testing.T) {
		_, err := run(StartInput{Template: "nope"})
		if codeOf(t, err) != "challenge_unknown" {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestReflect_OnlyOnceOver(t *testing.T) {
	couple := uuid.New()
	c := running(couple, onDay(3))
	repo := &fakeRepo{current: c}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{})

	if _, err := svc.Reflect(context.Background(), uuid.New(), c.ID, "Good."); codeOf(t, err) != "challenge_not_over" {
		t.Fatalf("err = %v, want challenge_not_over", err)
	}

	repo.current.Status = StatusEnded
	if _, err := svc.Reflect(context.Background(), uuid.New(), c.ID, "  Good.  "); err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if len(repo.reflects) != 1 || repo.reflects[0] != "Good." {
		t.Errorf("reflects = %q, want it trimmed", repo.reflects)
	}

	long := make([]rune, MaxReflectionRunes+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err := svc.Reflect(context.Background(), uuid.New(), c.ID, string(long))
	if ae, _ := apperr.As(err); ae == nil || ae.Fields["text"] == "" {
		t.Errorf("err = %v, want a field error on text", err)
	}
}

func TestLeave_GoesThroughEnd(t *testing.T) {
	couple := uuid.New()
	repo := &fakeRepo{current: running(couple, onDay(2))}
	svc := NewService(repo, fakeCouples{id: couple}, time.Now, &fakePoker{})
	if err := svc.Leave(context.Background(), uuid.New()); err != nil || repo.ended != 1 {
		t.Errorf("ended = %d, err = %v", repo.ended, err)
	}
}
