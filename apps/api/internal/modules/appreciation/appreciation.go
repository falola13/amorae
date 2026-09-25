// Package appreciation owns the short notes one partner sends the other.
// Personal, not social — no feed, like, reaction, or count.
package appreciation

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

const maxTextRunes = 500

// UndoWindow is how long the sender has to take a note back (BR-APPR-02).
// The client shows 5s of Undo; the extra 25s covers a slow connection, not indecision.
const UndoWindow = 30 * time.Second

var (
	ErrNotFound = apperr.NotFound("not_found", "That note isn’t here.")
	// Deliberately not a 404: the note is visible on screen, so pretending it doesn't exist would be stranger.
	ErrNotYours     = apperr.Forbidden("forbidden", "That note isn’t yours to take back.")
	ErrWindowClosed = apperr.Conflict("undo_window_closed", "Too late to take that one back — they may have read it already.")
)

type Appreciation struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	// No recipient field: a couple has two people, so it's whoever didn't send it (BR-APPR-01).
	FromID uuid.UUID
	Date   time.Time
	Text   string
	// To the instant; UndoWindow is measured from this, so a date alone wouldn't be precise enough.
	SentAt time.Time
}

// ValidateText's empty-note message matches the composer's own wording.
func ValidateText(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		return "", apperr.Validation(map[string]string{"text": "One true sentence is plenty."})
	case utf8.RuneCountInString(text) > maxTextRunes:
		return "", apperr.Validation(map[string]string{
			"text": fmt.Sprintf("Keep it under %d characters.", maxTextRunes),
		})
	}
	return text, nil
}

// CanUndo checks ownership first, so a non-owner never learns whether the window happened to be open.
func CanUndo(a Appreciation, userID uuid.UUID, now time.Time) error {
	if a.FromID != userID {
		return ErrNotYours
	}
	if now.Sub(a.SentAt) > UndoWindow {
		return ErrWindowClosed
	}
	return nil
}
