package events

import (
	"strings"
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func ptr[T any](v T) *T { return &v }

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	appErr, ok := err.(*apperr.Error)
	if !ok {
		t.Fatalf("err = %T, want *apperr.Error", err)
	}
	return appErr.Fields
}

func TestValidate_Creating(t *testing.T) {
	t.Run("a title and a date are enough", func(t *testing.T) {
		e, err := Event{}.Validate(Input{
			Title: ptr("Dinner at the place we liked"),
			Date:  ptr("2026-10-02"),
		}, true)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if e.Title != "Dinner at the place we liked" {
			t.Errorf("title = %q", e.Title)
		}
		if e.Date.Format(time.DateOnly) != "2026-10-02" {
			t.Errorf("date = %s", e.Date)
		}
	})

	t.Run("without them, both are reported at once", func(t *testing.T) {
		fields := fieldsOf(t, func() error { _, err := Event{}.Validate(Input{}, true); return err }())
		if fields["title"] == "" || fields["date"] == "" {
			t.Errorf("fields = %v, want both title and date", fields)
		}
	})

	t.Run("a date that is not a date", func(t *testing.T) {
		for _, bad := range []string{"tomorrow", "2026-13-01", "02/10/2026", "2026-2-1"} {
			_, err := Event{}.Validate(Input{Title: ptr("x"), Date: ptr(bad)}, true)
			if err == nil {
				t.Errorf("Validate accepted date %q", bad)
			}
		}
	})
}

func TestValidate_Editing(t *testing.T) {
	existing := Event{
		Title:     "Dinner",
		Date:      time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		Location:  "The place we liked",
		StartTime: "19:00",
	}

	t.Run("a field nobody mentioned is left alone", func(t *testing.T) {
		e, err := existing.Validate(Input{Title: ptr("Late dinner")}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if e.Title != "Late dinner" {
			t.Errorf("title = %q", e.Title)
		}
		if e.Location != existing.Location || e.StartTime != existing.StartTime {
			t.Error("a field that was not mentioned changed")
		}
		if !e.Date.Equal(existing.Date) {
			t.Error("the date changed without being mentioned")
		}
	})

	t.Run("an empty string is how a field is cleared", func(t *testing.T) {
		e, err := existing.Validate(Input{Location: ptr(""), StartTime: ptr("")}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if e.Location != "" || e.StartTime != "" {
			t.Errorf("location = %q, start = %q, want both empty", e.Location, e.StartTime)
		}
	})

	t.Run("an edit does not have to carry the required fields", func(t *testing.T) {
		if _, err := existing.Validate(Input{Notes: ptr("bring the tickets")}, false); err != nil {
			t.Errorf("Validate: %v", err)
		}
	})
}

func TestValidate_Times(t *testing.T) {
	base := Event{Title: "x", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}

	t.Run("an end before its start is refused", func(t *testing.T) {
		fields := fieldsOf(t, func() error {
			_, err := base.Validate(Input{StartTime: ptr("19:00"), EndTime: ptr("18:00")}, false)
			return err
		}())
		if fields["end_time"] == "" {
			t.Errorf("fields = %v, want end_time", fields)
		}
	})

	t.Run("the same time twice is fine", func(t *testing.T) {
		if _, err := base.Validate(Input{StartTime: ptr("19:00"), EndTime: ptr("19:00")}, false); err != nil {
			t.Errorf("Validate: %v", err)
		}
	})

	t.Run("a time that is not a time", func(t *testing.T) {
		for _, bad := range []string{"7pm", "25:00", "19:60", "7:00"} {
			_, err := base.Validate(Input{StartTime: ptr(bad)}, false)
			if err == nil {
				t.Errorf("Validate accepted start_time %q", bad)
			}
		}
	})
}

func TestValidate_Checklist(t *testing.T) {
	base := Event{Title: "x", Date: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}

	t.Run("the order given is the order kept", func(t *testing.T) {
		e, err := base.Validate(Input{Checklist: &[]string{"Book it", "Ask Ada", "Petrol"}}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		for i, want := range []string{"Book it", "Ask Ada", "Petrol"} {
			if e.Checklist[i].Text != want || e.Checklist[i].Position != i {
				t.Errorf("item %d = %+v", i, e.Checklist[i])
			}
		}
	})

	t.Run("blank rows are dropped, not complained about", func(t *testing.T) {
		e, err := base.Validate(Input{Checklist: &[]string{"Book it", "   ", "", "Petrol"}}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(e.Checklist) != 2 {
			t.Fatalf("%d items, want 2", len(e.Checklist))
		}
		// Positions close up: a hole would later be rejected by the unique index.
		if e.Checklist[1].Position != 1 {
			t.Errorf("second item has position %d", e.Checklist[1].Position)
		}
	})

	t.Run("a list is capped", func(t *testing.T) {
		many := make([]string, MaxChecklistItems+1)
		for i := range many {
			many[i] = "something"
		}
		fields := fieldsOf(t, func() error {
			_, err := base.Validate(Input{Checklist: &many}, false)
			return err
		}())
		if fields["checklist"] == "" {
			t.Errorf("fields = %v, want checklist", fields)
		}
	})

	t.Run("an empty list clears it", func(t *testing.T) {
		withItems := base
		withItems.Checklist = []ChecklistItem{{Text: "Book it"}}
		e, err := withItems.Validate(Input{Checklist: &[]string{}}, false)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(e.Checklist) != 0 {
			t.Errorf("%d items survived", len(e.Checklist))
		}
	})
}

func TestValidate_ReportsEverythingAtOnce(t *testing.T) {
	fields := fieldsOf(t, func() error {
		_, err := Event{}.Validate(Input{
			Title:    ptr(strings.Repeat("x", maxTitleRunes+1)),
			Date:     ptr("nope"),
			Location: ptr(strings.Repeat("y", maxLocationRunes+1)),
		}, true)
		return err
	}())
	for _, want := range []string{"title", "date", "location"} {
		if fields[want] == "" {
			t.Errorf("%s was not reported: %v", want, fields)
		}
	}
}
