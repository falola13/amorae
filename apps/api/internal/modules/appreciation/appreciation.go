// Package appreciation owns the short notes one partner sends the other.
//
// Personal, not social. There is no feed, no like, no reaction and no count,
// and nothing in here should ever grow one: the moment a note has an audience
// it stops being a thing you write to one person.
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
//
// The client offers Undo for five seconds. This is thirty, and the extra
// twenty-five are for the connection, not the person: a tap at 4.9 seconds on
// a bad line should still land. Beyond that the note is read, or may as well
// be, and taking it back would be editing somebody else's memory of it.
const UndoWindow = 30 * time.Second

var (
	ErrNotFound = apperr.NotFound("not_found", "That note isn’t here.")
	// The partner's note, not yours. Deliberately not a 404: you can see it
	// on the screen, and pretending it does not exist would be stranger than
	// saying no.
	ErrNotYours = apperr.Forbidden("forbidden", "That note isn’t yours to take back.")
	// Too late. Named for what happened rather than what was refused.
	ErrWindowClosed = apperr.Conflict("undo_window_closed", "Too late to take that one back — they may have read it already.")
)

// Appreciation is one note, from one of them to the other.
type Appreciation struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	// Who sent it. There is no recipient: a couple has two people, so it is
	// the other one (BR-APPR-01).
	FromID uuid.UUID
	Date   time.Time
	Text   string
	// When it was sent, to the instant. The undo window is measured from
	// this, so a date would not be precise enough to answer with.
	SentAt time.Time
}

// ValidateText bounds a note. The message for an empty one is the composer's
// own, so somebody who sees it recognises where it came from.
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

// CanUndo reports whether this person may still take this note back.
//
// Whose it is comes first. Somebody asking about a note they did not send
// should be told no for that reason, and never learn from the answer whether
// its window happened to be open.
func CanUndo(a Appreciation, userID uuid.UUID, now time.Time) error {
	if a.FromID != userID {
		return ErrNotYours
	}
	if now.Sub(a.SentAt) > UndoWindow {
		return ErrWindowClosed
	}
	return nil
}
