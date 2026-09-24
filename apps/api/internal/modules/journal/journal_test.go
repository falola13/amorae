package journal

import (
	"strings"
	"testing"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestValidate(t *testing.T) {
	t.Run("a tag from the picker and a line of text", func(t *testing.T) {
		tag, text, err := Validate("Gratitude", "  For the quiet morning.  ")
		if err != nil {
			t.Fatalf("a good entry was refused: %v", err)
		}
		if tag != TagGratitude {
			t.Errorf("tag = %q", tag)
		}
		if text != "For the quiet morning." {
			t.Errorf("text = %q, want it trimmed", text)
		}
	})

	t.Run("every tag the composer offers is accepted", func(t *testing.T) {
		for _, tag := range tags {
			if _, _, err := Validate(string(tag), "Something."); err != nil {
				t.Errorf("%q was refused: %v", tag, err)
			}
		}
	})

	t.Run("a tag we never offered is not", func(t *testing.T) {
		// It comes from a picker, so a different value means a client sending
		// something this server has never shown anybody.
		if _, _, err := Validate("Prayer reflection", "Something."); err == nil {
			t.Error("an unknown tag was accepted")
		}
	})

	t.Run("and neither is a different spelling of one", func(t *testing.T) {
		if _, _, err := Validate("gratitude", "Something."); err == nil {
			t.Error("a lowercase tag was accepted, so two spellings can now exist")
		}
	})

	t.Run("an entry needs words", func(t *testing.T) {
		if _, _, err := Validate("Gratitude", "   "); err == nil {
			t.Error("a blank entry was kept")
		}
	})

	t.Run("there is room to actually write", func(t *testing.T) {
		if _, _, err := Validate("Reflection", strings.Repeat("a", maxTextRunes)); err != nil {
			t.Errorf("an entry of exactly the limit was refused: %v", err)
		}
		if _, _, err := Validate("Reflection", strings.Repeat("a", maxTextRunes+1)); err == nil {
			t.Error("an oversized entry was kept")
		}
	})

	t.Run("both problems are reported together", func(t *testing.T) {
		// One round trip should tell somebody everything that is wrong.
		_, _, err := Validate("Nonsense", "")
		if err == nil {
			t.Fatal("nothing was wrong with a bad tag and no text")
		}
		appErr, ok := apperr.As(err)
		if !ok {
			t.Fatalf("err = %v, want a validation error", err)
		}
		if appErr.Fields["tag"] == "" || appErr.Fields["text"] == "" {
			t.Errorf("fields = %v, want both named", appErr.Fields)
		}
	})
}
