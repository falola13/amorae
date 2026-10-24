package anniversary

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func date(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr(t time.Time) *time.Time { return &t }

func TestNext(t *testing.T) {
	cases := []struct {
		name, start, today string
		wantOK             bool
		wantOn             string
		wantYears, wantIn  int
	}{
		{"tomorrow", "2023-10-25", "2026-10-24", true, "2026-10-25", 3, 1},
		{"today", "2023-10-25", "2026-10-25", true, "2026-10-25", 3, 0},
		{"just gone is next year's", "2023-10-25", "2026-10-26", true, "2027-10-25", 4, 364},
		{"not a year in yet", "2026-03-01", "2026-10-24", true, "2027-03-01", 1, 128},
		{"started this very day", "2026-10-24", "2026-10-24", false, "", 0, 0},
		{"a leap day, in a short year", "2024-02-29", "2027-02-20", true, "2027-02-28", 3, 8},
		{"a leap day, in a leap year", "2024-02-29", "2028-02-20", true, "2028-02-29", 4, 9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, ok := Next(ptr(date(tc.start)), date(tc.today))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got := o.On.Format(time.DateOnly); got != tc.wantOn {
				t.Errorf("on = %s, want %s", got, tc.wantOn)
			}
			if o.Years != tc.wantYears || o.DaysUntil != tc.wantIn {
				t.Errorf("years %d in %d, want %d in %d", o.Years, o.DaysUntil, tc.wantYears, tc.wantIn)
			}
		})
	}

	t.Run("no start date, no anniversary", func(t *testing.T) {
		if _, ok := Next(nil, date("2026-10-24")); ok {
			t.Error("an anniversary was made up")
		}
	})
}

func TestLocalToday(t *testing.T) {
	// 23:30 UTC on the 24th is already the 25th in Lagos.
	now := time.Date(2026, 10, 24, 23, 30, 0, 0, time.UTC)
	if got := LocalToday(now, "Africa/Lagos").Format(time.DateOnly); got != "2026-10-25" {
		t.Errorf("Lagos today = %s, want 2026-10-25", got)
	}
	if got := LocalToday(now, "nowhere/at-all").Format(time.DateOnly); got != "2026-10-24" {
		t.Errorf("an unknown zone = %s, want UTC's 2026-10-24", got)
	}
}

func TestWritable(t *testing.T) {
	if !(Occasion{DaysUntil: WriteAhead}).Writable() {
		t.Error("the first day of the window was closed")
	}
	if (Occasion{DaysUntil: WriteAhead + 1}).Writable() {
		t.Error("a letter could be written before the window opened")
	}
	if !(Occasion{DaysUntil: 0}).Writable() {
		t.Error("the day itself was closed")
	}
}

func TestYearBefore(t *testing.T) {
	o := Occasion{On: date("2028-02-29")}
	if got := o.YearBefore().Format(time.DateOnly); got != "2027-02-28" {
		t.Errorf("year before = %s, want 2027-02-28", got)
	}
}

func TestAssemble_KeepsTheSeal(t *testing.T) {
	me, them := uuid.New(), uuid.New()
	on := date("2026-10-25")
	o := Occasion{On: on, Years: 3, DaysUntil: 1}
	letters := []Letter{
		{AuthorID: them, OpensOn: on, Text: "For tomorrow"},
		{AuthorID: me, OpensOn: on, Text: "Mine"},
		{AuthorID: them, OpensOn: date("2025-10-25"), Text: "Last year's"},
		{AuthorID: me, OpensOn: date("2025-10-25"), Text: "My last year's"},
	}

	t.Run("the day before: theirs is sealed, mine is mine", func(t *testing.T) {
		v := assemble(me, o, date("2026-10-24"), letters, Year{})
		if v.Theirs != nil {
			t.Fatalf("their letter was handed over a day early: %q", v.Theirs.Text)
		}
		if !v.TheirsSealed {
			t.Error("a waiting letter wasn't said to be waiting")
		}
		if v.Mine == nil || v.Mine.Text != "Mine" {
			t.Errorf("mine = %+v", v.Mine)
		}
		if len(v.Past) != 2 {
			t.Errorf("past = %d letters, want both of last year's", len(v.Past))
		}
	})

	t.Run("on the day: it opens", func(t *testing.T) {
		v := assemble(me, Occasion{On: on, Years: 3}, on, letters, Year{})
		if v.Theirs == nil || v.Theirs.Text != "For tomorrow" {
			t.Fatalf("theirs = %+v, want it open", v.Theirs)
		}
		if v.TheirsSealed {
			t.Error("an open letter was still called sealed")
		}
	})

	t.Run("nothing written is nothing waiting", func(t *testing.T) {
		v := assemble(me, o, date("2026-10-24"), nil, Year{})
		if v.Theirs != nil || v.TheirsSealed || v.Mine != nil || v.Past == nil {
			t.Errorf("view = %+v", v)
		}
	})
}

func TestValidateLetter(t *testing.T) {
	if got, err := ValidateLetter("  Dear you  "); err != nil || got != "Dear you" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := ValidateLetter("  "); err == nil {
		t.Error("an empty letter was kept")
	}
	if _, err := ValidateLetter(strings.Repeat("é", maxLetterRunes)); err != nil {
		t.Errorf("a letter at the limit was refused: %v", err)
	}
	if _, err := ValidateLetter(strings.Repeat("a", maxLetterRunes+1)); err == nil {
		t.Error("an oversized letter was kept")
	}
}
