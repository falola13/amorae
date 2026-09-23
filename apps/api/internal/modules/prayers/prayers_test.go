package prayers

import (
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
			// 00:30 Sunday in Lagos is still Saturday evening in New York, so a
			// couple keeping New York time is in the previous week. This is the
			// whole reason the couple's timezone decides, not the server's.
			name: "the same instant can be different weeks in different zones",
			at:   time.Date(2026, 9, 20, 0, 30, 0, 0, lagos),
			loc:  newYork,
			want: date(2026, 9, 13),
		},
		{
			// New York moves to standard time on 1 November 2026; the Sunday
			// boundary must not drift by an hour.
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

	t.Run("the partner is told a week is coming, but sees nothing in it", func(t *testing.T) {
		if got := StatusFor(draft, partner); got != StatusWaiting {
			t.Errorf("StatusFor() = %q, want %q", got, StatusWaiting)
		}
		if got := PointsFor(draft, partner); got != nil {
			t.Errorf("PointsFor() returned %d points to the waiting partner", len(got))
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

	t.Run("once published, both see the same thing", func(t *testing.T) {
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

func TestCanEditPoints(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	week := Week{SetterUserID: setter, Status: StatusDraft}

	if err := CanEditPoints(week, setter, false); err != nil {
		t.Errorf("the setter should be able to edit their own draft: %v", err)
	}
	if err := CanEditPoints(week, partner, false); err == nil {
		t.Error("the other partner must not edit the week")
	}
	// The point of the rule: nobody rewrites prayers someone is already praying.
	if err := CanEditPoints(week, setter, true); err == nil {
		t.Error("editing should be refused once the partner has completed any point")
	}
}

func TestCanPublish(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	week := Week{SetterUserID: setter, Status: StatusDraft}

	if err := CanPublish(week, setter); err != nil {
		t.Errorf("the setter should be able to publish: %v", err)
	}
	if err := CanPublish(week, partner); err == nil {
		t.Error("only the setter publishes the week")
	}
}

// errorOf runs fn, requires it to fail, and hands back the typed error so a
// test can look at its fields.
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
