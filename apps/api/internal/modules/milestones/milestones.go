// Package milestones owns the dates a couple keeps: birthdays, anniversaries,
// the day they met. One entity, not one per kind (BR-DATE-01). A row is the
// day it happened; "when is this next" is worked out by callers, not stored.
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

type Milestone struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	Title    string
	Date     time.Time
	Sub      string
	// Whether this one comes round for them every year, or is simply kept.
	Reminder bool
}

type Input struct {
	Title    string
	Date     time.Time
	Sub      string
	Reminder bool
}

// Validate trims and bounds fields using the same wording as the composer, so errors are recognizable.
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
