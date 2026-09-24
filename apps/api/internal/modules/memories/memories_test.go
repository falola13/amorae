package memories

import (
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)

	t.Run("it keeps what was written, trimmed", func(t *testing.T) {
		in, err := Validate(Input{
			Title:    "  Our first Amorae date night  ",
			Date:     day,
			Location: " Lekki ",
			Note:     "  We stayed until they stacked the chairs.  ",
		})
		if err != nil {
			t.Fatalf("a good moment was refused: %v", err)
		}
		if in.Title != "Our first Amorae date night" || in.Location != "Lekki" {
			t.Errorf("title = %q, location = %q", in.Title, in.Location)
		}
		if !strings.HasSuffix(in.Note, "chairs.") {
			t.Errorf("note = %q", in.Note)
		}
	})

	t.Run("a moment needs something that happened", func(t *testing.T) {
		if _, err := Validate(Input{Title: "  ", Date: day}); err == nil {
			t.Error("a blank moment was kept")
		}
	})

	t.Run("and a day it happened on", func(t *testing.T) {
		if _, err := Validate(Input{Title: "Something"}); err == nil {
			t.Error("a moment with no day was kept")
		}
	})

	t.Run("the note has more room than the title", func(t *testing.T) {
		// It is the part worth writing: 500 characters against 80.
		if _, err := Validate(Input{
			Title: "Fine", Date: day, Note: strings.Repeat("a", maxNoteRunes),
		}); err != nil {
			t.Errorf("a note of exactly the limit was refused: %v", err)
		}
		if _, err := Validate(Input{
			Title: "Fine", Date: day, Note: strings.Repeat("a", maxNoteRunes+1),
		}); err == nil {
			t.Error("an oversized note was kept")
		}
	})

	t.Run("length is counted in characters, not bytes", func(t *testing.T) {
		if _, err := Validate(Input{Title: strings.Repeat("é", maxTitleRunes), Date: day}); err != nil {
			t.Errorf("a title of exactly the limit was refused: %v", err)
		}
	})

	t.Run("where it was is optional", func(t *testing.T) {
		if _, err := Validate(Input{Title: "Something", Date: day}); err != nil {
			t.Errorf("a moment with no place was refused: %v", err)
		}
	})
}
