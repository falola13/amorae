package appreciation

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateText(t *testing.T) {
	t.Run("it trims", func(t *testing.T) {
		got, err := ValidateText("  You made today easier.  ")
		if err != nil || got != "You made today easier." {
			t.Errorf("got %q, err %v", got, err)
		}
	})

	t.Run("a note needs words", func(t *testing.T) {
		if _, err := ValidateText("   "); err == nil {
			t.Error("an empty note was sent")
		}
	})

	t.Run("and stays short", func(t *testing.T) {
		if _, err := ValidateText(strings.Repeat("a", maxTextRunes)); err != nil {
			t.Errorf("a note of exactly the limit was refused: %v", err)
		}
		if _, err := ValidateText(strings.Repeat("a", maxTextRunes+1)); err == nil {
			t.Error("an oversized note was sent")
		}
	})
}

func TestCanUndo(t *testing.T) {
	sender, partner := uuid.New(), uuid.New()
	sent := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	note := Appreciation{FromID: sender, SentAt: sent}

	t.Run("the sender, straight away", func(t *testing.T) {
		if err := CanUndo(note, sender, sent.Add(time.Second)); err != nil {
			t.Errorf("the sender could not take back what they just sent: %v", err)
		}
	})

	t.Run("and at the last moment of the window", func(t *testing.T) {
		if err := CanUndo(note, sender, sent.Add(UndoWindow)); err != nil {
			t.Errorf("a tap inside the window was refused: %v", err)
		}
	})

	t.Run("but not after it", func(t *testing.T) {
		err := CanUndo(note, sender, sent.Add(UndoWindow+time.Millisecond))
		if err != ErrWindowClosed {
			t.Errorf("err = %v, want the window closed", err)
		}
	})

	t.Run("never the partner's note", func(t *testing.T) {
		if err := CanUndo(note, partner, sent.Add(time.Second)); err != ErrNotYours {
			t.Errorf("err = %v, want it refused as not theirs", err)
		}
	})

	t.Run("whose it is is decided before when it was", func(t *testing.T) {
		err := CanUndo(note, partner, sent.Add(time.Hour))
		if err != ErrNotYours {
			t.Errorf("err = %v, want not-theirs rather than a timing answer", err)
		}
	})
}
