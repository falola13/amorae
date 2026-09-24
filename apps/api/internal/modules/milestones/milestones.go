// Package milestones owns the dates a couple keeps: birthdays, anniversaries,
// the day they met. One entity rather than one per kind (BR-DATE-01).
//
// A date here is the day something happened, never the next time it comes
// round. Everything that wants "when is this next" works it out — the list
// screen so it can show what is coming up, the notification worker so it can
// say something on the morning. That keeps one fact in the database and no
// January where every row is wrong.
package milestones

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var ErrNotFound = apperr.NotFound("not_found", "That date isn’t here.")

const (
	maxTitleRunes = 80
	maxSubRunes   = 80
)

// Milestone is one kept date.
type Milestone struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	Title    string
	Date     time.Time
	Sub      string
	// Whether this one comes round for them every year, or is simply kept.
	Reminder bool
}

// Input is what either partner sends to add one.
type Input struct {
	Title    string
	Date     time.Time
	Sub      string
	Reminder bool
}

// Validate trims and bounds an input, in the same words the composer uses so
// a person who sees an error recognises the field it belongs to.
func Validate(in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Sub = strings.TrimSpace(in.Sub)

	fields := map[string]string{}
	switch {
	case in.Title == "":
		fields["title"] = "What is it?"
	case utf8.RuneCountInString(in.Title) > maxTitleRunes:
		fields["title"] = "Keep it short."
	}
	if utf8.RuneCountInString(in.Sub) > maxSubRunes {
		fields["sub"] = "Keep it short."
	}
	if in.Date.IsZero() {
		fields["date"] = "Pick a date."
	}
	if len(fields) > 0 {
		return Input{}, apperr.Validation(fields)
	}
	return in, nil
}
