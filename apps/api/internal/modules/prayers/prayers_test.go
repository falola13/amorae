package prayers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	return loc
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestStartOfWeek_IsTheCouplesLocalSunday(t *testing.T) {
	lagos := mustLoad(t, "Africa/Lagos") // UTC+1, no daylight saving
	newYork := mustLoad(t, "America/New_York")

	tests := []struct {
		name string
		at   time.Time
		loc  *time.Location
		want time.Time
	}{
		{
			name: "midweek",
			at:   time.Date(2026, 9, 23, 15, 4, 0, 0, time.UTC), // Wednesday
			loc:  lagos,
			want: date(2026, 9, 20),
		},
		{
			name: "the Sunday itself counts as its own week",
			at:   time.Date(2026, 9, 20, 0, 30, 0, 0, lagos),
			loc:  lagos,
			want: date(2026, 9, 20),
		},
		{
			name: "a moment before the local Sunday still belongs to the week before",
			at:   time.Date(2026, 9, 19, 23, 59, 0, 0, lagos),
			loc:  lagos,
			want: date(2026, 9, 13),
		},
		{
			// 00:30 Sunday in Lagos is still Saturday evening in New York.
			name: "the same instant can be different weeks in different zones",
			at:   time.Date(2026, 9, 20, 0, 30, 0, 0, lagos),
			loc:  newYork,
			want: date(2026, 9, 13),
		},
		{
			// New York moves to standard time on 1 November 2026.
			name: "daylight saving does not move the boundary",
			at:   time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC),
			loc:  newYork,
			want: date(2026, 11, 1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StartOfWeek(tc.at, tc.loc); !got.Equal(tc.want) {
				t.Errorf("StartOfWeek() = %s, want %s", got.Format(time.DateOnly), tc.want.Format(time.DateOnly))
			}
		})
	}
}

func TestWeekIndex_CountsWholeWeeks(t *testing.T) {
	first := date(2026, 9, 6)
	for weeks := 0; weeks < 60; weeks++ {
		got := WeekIndex(first, first.AddDate(0, 0, 7*weeks))
		if got != weeks {
			t.Fatalf("WeekIndex() = %d, want %d", got, weeks)
		}
	}
}

func TestSetterFor_AlternatesInJoinOrder(t *testing.T) {
	first := Member{UserID: uuid.New(), JoinedAt: time.Date(2026, 2, 12, 9, 0, 0, 0, time.UTC)}
	second := Member{UserID: uuid.New(), JoinedAt: time.Date(2026, 2, 12, 11, 30, 0, 0, time.UTC)}

	t.Run("the first member sets the couple's first week", func(t *testing.T) {
		got, err := SetterFor([]Member{first, second}, 0)
		if err != nil {
			t.Fatalf("SetterFor() returned an error: %v", err)
		}
		if got != first.UserID {
			t.Error("week 0 should belong to whoever joined first")
		}
	})

	t.Run("then it alternates", func(t *testing.T) {
		for week := 0; week < 8; week++ {
			want := first.UserID
			if week%2 == 1 {
				want = second.UserID
			}
			got, err := SetterFor([]Member{first, second}, week)
			if err != nil {
				t.Fatalf("SetterFor() returned an error: %v", err)
			}
			if got != want {
				t.Errorf("week %d went to the wrong partner", week)
			}
		}
	})

	t.Run("the order of the slice does not decide it", func(t *testing.T) {
		forwards, err := SetterFor([]Member{first, second}, 3)
		if err != nil {
			t.Fatalf("SetterFor() returned an error: %v", err)
		}
		backwards, err := SetterFor([]Member{second, first}, 3)
		if err != nil {
			t.Fatalf("SetterFor() returned an error: %v", err)
		}
		if forwards != backwards {
			t.Error("whoever joined first anchors the rotation, whatever order the rows arrive in")
		}
	})

	t.Run("a couple needs two members", func(t *testing.T) {
		if _, err := SetterFor([]Member{first}, 0); err == nil {
			t.Error("SetterFor() with one member should fail rather than guess")
		}
	})
}

func TestValidatePoints(t *testing.T) {
	t.Run("keeps the given order as the position", func(t *testing.T) {
		got, err := ValidatePoints([]Point{{Title: "Work"}, {Title: "Family"}, {Title: "Rest"}})
		if err != nil {
			t.Fatalf("ValidatePoints() returned an error: %v", err)
		}
		for i, p := range got {
			if p.Position != i {
				t.Errorf("point %d has position %d", i, p.Position)
			}
		}
	})

	t.Run("trims, and treats whitespace as empty", func(t *testing.T) {
		got, err := ValidatePoints([]Point{{Title: "  Work  ", Body: "  for the week  "}})
		if err != nil {
			t.Fatalf("ValidatePoints() returned an error: %v", err)
		}
		if got[0].Title != "Work" || got[0].Body != "for the week" {
			t.Errorf("got %+v, want trimmed values", got[0])
		}
		if _, err := ValidatePoints([]Point{{Title: "   "}}); err == nil {
			t.Error("a title of spaces should be rejected")
		}
	})

	t.Run("a title alone is a complete point", func(t *testing.T) {
		if _, err := ValidatePoints([]Point{{Title: "Ada’s interview"}}); err != nil {
			t.Errorf("ValidatePoints() rejected a title-only point: %v", err)
		}
	})

	t.Run("caps the week at MaxPoints", func(t *testing.T) {
		tooMany := make([]Point, MaxPoints+1)
		for i := range tooMany {
			tooMany[i] = Point{Title: "Something"}
		}
		err := errorOf(t, func() error { _, e := ValidatePoints(tooMany); return e })
		if err.Fields["points"] == "" {
			t.Errorf("fields = %v, want a message under points", err.Fields)
		}

		exactly := tooMany[:MaxPoints]
		if _, err := ValidatePoints(exactly); err != nil {
			t.Errorf("%d points should be allowed: %v", MaxPoints, err)
		}
	})

	t.Run("reports every bad point at once, keyed by its position", func(t *testing.T) {
		err := errorOf(t, func() error {
			_, e := ValidatePoints([]Point{
				{Title: "Fine"},
				{Title: ""},
				{Title: strings.Repeat("x", maxTitleRunes+1)},
			})
			return e
		})
		if err.Fields["points.1.title"] == "" || err.Fields["points.2.title"] == "" {
			t.Errorf("fields = %v, want both bad points reported", err.Fields)
		}
		if _, reported := err.Fields["points.0.title"]; reported {
			t.Error("the valid point should not be reported")
		}
	})

	t.Run("counts characters, not bytes", func(t *testing.T) {
		// Emoji are 4 bytes each: counting bytes would reject a legitimate title.
		if _, err := ValidatePoints([]Point{{Title: strings.Repeat("🙏", maxTitleRunes)}}); err != nil {
			t.Errorf("a title of %d characters was rejected: %v", maxTitleRunes, err)
		}
	})
}

func TestViewOfADraftWeek(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	draft := Week{
		SetterUserID: setter,
		Status:       StatusDraft,
		Points:       []Point{{Title: "Still writing this"}},
	}

	t.Run("the partner sees the draft too, since either may step in", func(t *testing.T) {
		if got := StatusFor(draft, partner); got != StatusDraft {
			t.Errorf("StatusFor() = %q, want %q", got, StatusDraft)
		}
		if got := PointsFor(draft, partner); len(got) != 1 {
			t.Errorf("PointsFor() returned %d points to the partner, want 1", len(got))
		}
	})

	t.Run("the setter sees their own draft", func(t *testing.T) {
		if got := StatusFor(draft, setter); got != StatusDraft {
			t.Errorf("StatusFor() = %q, want %q", got, StatusDraft)
		}
		if got := PointsFor(draft, setter); len(got) != 1 {
			t.Errorf("PointsFor() returned %d points to the setter, want 1", len(got))
		}
	})

	t.Run("once published, both still see the same thing", func(t *testing.T) {
		published := draft
		published.Status = StatusPublished
		for _, viewer := range []uuid.UUID{setter, partner} {
			if got := StatusFor(published, viewer); got != StatusPublished {
				t.Errorf("StatusFor() = %q, want %q", got, StatusPublished)
			}
			if len(PointsFor(published, viewer)) != 1 {
				t.Error("both partners should see the points of a published week")
			}
		}
	})
}

func TestScheduledOn(t *testing.T) {
	sunday := time.Sunday
	wednesday := time.Wednesday

	t.Run("every day matches every weekday", func(t *testing.T) {
		for day := time.Sunday; day <= time.Saturday; day++ {
			if !ScheduledOn(AllWeekdays, day) {
				t.Errorf("ScheduledOn(AllWeekdays, %s) = false", day)
			}
		}
	})

	t.Run("a single day matches only itself", func(t *testing.T) {
		mask := 1 << uint(wednesday)
		if !ScheduledOn(mask, wednesday) {
			t.Error("ScheduledOn() = false for the day it names")
		}
		if ScheduledOn(mask, sunday) {
			t.Error("ScheduledOn() = true for a day it doesn't name")
		}
	})

	t.Run("several days matches each of them and nothing else", func(t *testing.T) {
		mask := 1<<uint(time.Monday) | 1<<uint(time.Friday)
		for day := time.Sunday; day <= time.Saturday; day++ {
			want := day == time.Monday || day == time.Friday
			if got := ScheduledOn(mask, day); got != want {
				t.Errorf("ScheduledOn(Mon+Fri, %s) = %v, want %v", day, got, want)
			}
		}
	})
}

func TestValidatePoints_Weekdays(t *testing.T) {
	t.Run("no weekdays given means every day", func(t *testing.T) {
		got, err := ValidatePoints([]Point{{Title: "Work"}})
		if err != nil {
			t.Fatalf("ValidatePoints() returned an error: %v", err)
		}
		if got[0].Weekdays != AllWeekdays {
			t.Errorf("Weekdays = %d, want %d (every day)", got[0].Weekdays, AllWeekdays)
		}
	})

	t.Run("a chosen set becomes the matching bitmask", func(t *testing.T) {
		got, err := ValidatePoints([]Point{{Title: "Work", WeekdaysRaw: []int{1, 3, 5}}})
		if err != nil {
			t.Fatalf("ValidatePoints() returned an error: %v", err)
		}
		want := 1<<1 | 1<<3 | 1<<5
		if got[0].Weekdays != want {
			t.Errorf("Weekdays = %d, want %d", got[0].Weekdays, want)
		}
	})

	t.Run("duplicates and order don't matter", func(t *testing.T) {
		got, err := ValidatePoints([]Point{{Title: "Work", WeekdaysRaw: []int{5, 1, 5, 1}}})
		if err != nil {
			t.Fatalf("ValidatePoints() returned an error: %v", err)
		}
		want := 1<<1 | 1<<5
		if got[0].Weekdays != want {
			t.Errorf("Weekdays = %d, want %d", got[0].Weekdays, want)
		}
	})

	t.Run("a day outside 0..6 is rejected", func(t *testing.T) {
		err := errorOf(t, func() error {
			_, e := ValidatePoints([]Point{{Title: "Work", WeekdaysRaw: []int{7}}})
			return e
		})
		if err.Fields["points.0.weekdays"] == "" {
			t.Errorf("fields = %v, want a message under points.0.weekdays", err.Fields)
		}
	})
}

func TestCanEditPoints(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	first, second := uuid.New(), uuid.New()
	week := Week{
		SetterUserID: setter,
		Status:       StatusPublished,
		Points: []Point{
			{ID: first, Position: 0, Title: "For his new job", Body: "That it settles."},
			{ID: second, Position: 1, Title: "For her mother"},
		},
	}
	prayed := map[uuid.UUID]bool{first: true}

	t.Run("either partner may edit, unlocked points aside", func(t *testing.T) {
		if err := CanEditPoints(week, setter, week.Points, nil); err != nil {
			t.Errorf("the setter should be able to edit the week: %v", err)
		}
		if err := CanEditPoints(week, partner, week.Points, nil); err != nil {
			t.Errorf("the partner should be able to edit the week too (DEC-33): %v", err)
		}
	})

	unchanged := func() []Point { return append([]Point(nil), week.Points...) }

	t.Run("adding is always allowed", func(t *testing.T) {
		points := append(unchanged(), Point{Title: "For the move"})
		if err := CanEditPoints(week, setter, points, prayed); err != nil {
			t.Errorf("adding after the partner started was refused: %v", err)
		}
	})

	t.Run("a point nobody has reached is still the setter's", func(t *testing.T) {
		points := unchanged()
		points[1].Title = "For her mother's health"
		if err := CanEditPoints(week, setter, points, prayed); err != nil {
			t.Errorf("editing an unprayed point was refused: %v", err)
		}
	})

	t.Run("a prayed point cannot be reworded", func(t *testing.T) {
		points := unchanged()
		points[0].Body = "That he turns it down."
		if err := CanEditPoints(week, setter, points, prayed); err == nil {
			t.Error("a point the partner had prayed was rewritten under them")
		}
	})

	t.Run("a prayed point cannot be taken away", func(t *testing.T) {
		points := []Point{week.Points[1]}
		if err := CanEditPoints(week, setter, points, prayed); err == nil {
			t.Error("a point the partner had prayed was deleted under them")
		}
	})

	t.Run("a prayed point may be moved", func(t *testing.T) {
		// Position isn't part of what CanEditPoints locks.
		points := []Point{week.Points[1], week.Points[0]}
		points[0].Position, points[1].Position = 0, 1
		if err := CanEditPoints(week, setter, points, prayed); err != nil {
			t.Errorf("reordering was refused: %v", err)
		}
	})

	t.Run("their own praying does not tie their hands", func(t *testing.T) {
		points := unchanged()
		points[0].Title = "For the new job"
		if err := CanEditPoints(week, setter, points, nil); err != nil {
			t.Errorf("the editor was blocked by their own completion: %v", err)
		}
	})

	t.Run("the partner is locked out of a point they didn't pray, same as the setter would be", func(t *testing.T) {
		// prayedByOthers is computed relative to the editor (Record.PrayedByOthers),
		// so from the partner's side, "others" means the setter — the lock is
		// symmetric, not tied to who happens to be the setter.
		lockedForPartner := map[uuid.UUID]bool{first: true}
		points := unchanged()
		points[0].Title = "Something else"
		if err := CanEditPoints(week, partner, points, lockedForPartner); err == nil {
			t.Error("the partner rewrote a point the setter had prayed")
		}
	})
}

// errorOf runs fn, requires it to fail, and hands back the typed error.
func errorOf(t *testing.T, fn func() error) *apperr.Error {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	appErr, ok := err.(*apperr.Error)
	if !ok {
		t.Fatalf("expected *apperr.Error, got %T", err)
	}
	return appErr
}

func TestValidateAnswerNote_AllowsEmpty(t *testing.T) {
	got, err := ValidateAnswerNote("   ")
	if err != nil {
		t.Fatalf("ValidateAnswerNote(blank) error = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("ValidateAnswerNote(blank) = %q, want empty", got)
	}
}

func TestValidateAnswerNote_TrimsAndBounds(t *testing.T) {
	got, err := ValidateAnswerNote("  he started on Monday  ")
	if err != nil {
		t.Fatalf("ValidateAnswerNote() error = %v, want nil", err)
	}
	if got != "he started on Monday" {
		t.Errorf("ValidateAnswerNote() = %q, want it trimmed", got)
	}

	if _, err := ValidateAnswerNote(strings.Repeat("a", maxAnswerRunes+1)); err == nil {
		t.Error("ValidateAnswerNote(too long) error = nil, want a validation error")
	}
}

func TestCanAnswer_OnlyOnceShared(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status Status
		wantOK bool
	}{
		{"published week can be answered", StatusPublished, true},
		{"a draft cannot: the partner has not seen it", StatusDraft, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CanAnswer(Week{Status: tc.status})
			if tc.wantOK && err != nil {
				t.Errorf("CanAnswer(%s) = %v, want nil", tc.status, err)
			}
			if !tc.wantOK && !errors.Is(err, ErrNotShared) {
				t.Errorf("CanAnswer(%s) = %v, want ErrNotShared", tc.status, err)
			}
		})
	}
}

func TestToDTO_TodayDaysAndLocked(t *testing.T) {
	viewer, partner := uuid.New(), uuid.New()
	weekStart := date(2026, 9, 20) // a Sunday
	sundayOnly := 1 << uint(time.Sunday)

	everyDay := uuid.New()    // scheduled every day
	sundaysOnly := uuid.New() // scheduled Sunday only, and prayed by partner

	rec := Record{
		Week: Week{
			ID:           uuid.New(),
			WeekStart:    weekStart,
			SetterUserID: viewer,
			Status:       StatusPublished,
			Points: []Point{
				{ID: everyDay, Title: "Every day", Weekdays: AllWeekdays},
				{ID: sundaysOnly, Title: "Sundays", Weekdays: sundayOnly},
			},
		},
		Completed: map[uuid.UUID][]uuid.UUID{
			partner: {sundaysOnly},
		},
		ByDay: map[time.Time]map[uuid.UUID][]uuid.UUID{
			weekStart: {partner: {sundaysOnly}},
		},
		Reflections: map[uuid.UUID]string{},
	}

	t.Run("viewed as the current week", func(t *testing.T) {
		out := ToDTO(rec, viewer, partner, weekStart)
		if out.Today != weekStart.Format(time.DateOnly) {
			t.Errorf("today = %q, want %q", out.Today, weekStart.Format(time.DateOnly))
		}
		if len(out.MyCompleted) != 0 {
			t.Errorf("my_completed = %v, want none — the viewer hasn't prayed today", out.MyCompleted)
		}
		if len(out.PartnerCompleted) != 1 || out.PartnerCompleted[0] != sundaysOnly.String() {
			t.Errorf("partner_completed = %v, want [%s]", out.PartnerCompleted, sundaysOnly)
		}
		if len(out.Days) != 7 {
			t.Fatalf("%d days, want 7", len(out.Days))
		}
		if len(out.Days[0].Points) != 2 {
			t.Errorf("Sunday has %d scheduled points, want 2", len(out.Days[0].Points))
		}
		if len(out.Days[1].Points) != 1 {
			t.Errorf("Monday has %d scheduled points, want 1 (every-day only)", len(out.Days[1].Points))
		}
		if len(out.Locked) != 1 || out.Locked[0] != sundaysOnly.String() {
			t.Errorf("locked = %v, want [%s] (the partner already prayed it)", out.Locked, sundaysOnly)
		}
	})

	t.Run("viewed as history, today is empty and completions are the whole week's", func(t *testing.T) {
		out := ToDTO(rec, viewer, partner, weekStart.AddDate(0, 0, 30))
		if out.Today != "" {
			t.Errorf("today = %q, want empty for a week that isn't current", out.Today)
		}
		if len(out.PartnerCompleted) != 1 || out.PartnerCompleted[0] != sundaysOnly.String() {
			t.Errorf("partner_completed = %v, want the week's union [%s]", out.PartnerCompleted, sundaysOnly)
		}
	})

	t.Run("the partner's own view has nothing locked", func(t *testing.T) {
		out := ToDTO(rec, partner, viewer, weekStart)
		if len(out.Locked) != 0 {
			t.Errorf("locked = %v, want none — nobody but the partner has prayed anything", out.Locked)
		}
	})
}
