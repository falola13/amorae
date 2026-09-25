// Package goals owns the things a couple is working toward together. One
// shared total, never split — the log exists for history, not division
// (BR-GOAL-01). Nothing gamified.
package goals

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

const (
	maxTitleRunes = 80
	maxWhyRunes   = 200
	maxLabelRunes = 24
)

var ErrNotFound = apperr.NotFound("goal_not_found", "That goal isn’t here.")

type Unit string

const (
	UnitNaira Unit = "naira"
	UnitCount Unit = "count"
)

type Goal struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	Title     string
	Why       string
	Target    int64
	Unit      Unit
	UnitLabel string
	StartDate time.Time
	EndDate   time.Time
	Done      bool
	Progress  []Progress
}

type Progress struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Amount int64
	Date   time.Time
}

// Total is computed from the log each time, never stored — a stored counter could drift from it.
func (g Goal) Total() int64 {
	var total int64
	for _, p := range g.Progress {
		total += p.Amount
	}
	return total
}

// Input is a create or edit; a nil field is left alone.
type Input struct {
	Title     *string
	Why       *string
	Target    *int64
	Unit      *string
	UnitLabel *string
	StartDate *string
	EndDate   *string
	Done      *bool
}

// Validate lays an input over a goal and reports everything wrong at once.
func (g Goal) Validate(in Input, creating bool) (Goal, error) {
	fields := map[string]string{}

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		switch {
		case title == "":
			fields["title"] = "What would you like to build together?"
		case utf8.RuneCountInString(title) > maxTitleRunes:
			fields["title"] = fmt.Sprintf("Keep it under %d characters.", maxTitleRunes)
		default:
			g.Title = title
		}
	} else if creating {
		fields["title"] = "What would you like to build together?"
	}

	if in.Why != nil {
		why := strings.TrimSpace(*in.Why)
		if utf8.RuneCountInString(why) > maxWhyRunes {
			fields["why"] = fmt.Sprintf("Keep it under %d characters.", maxWhyRunes)
		} else {
			g.Why = why
		}
	}

	if in.Target != nil {
		if *in.Target <= 0 {
			fields["target"] = "Set a target above zero."
		} else {
			g.Target = *in.Target
		}
	} else if creating {
		fields["target"] = "Set a target above zero."
	}

	if in.Unit != nil {
		switch Unit(strings.TrimSpace(*in.Unit)) {
		case UnitNaira:
			g.Unit = UnitNaira
		case UnitCount:
			g.Unit = UnitCount
		default:
			fields["unit"] = "Choose naira or a count."
		}
	} else if creating {
		fields["unit"] = "Choose naira or a count."
	}

	if in.UnitLabel != nil {
		label := strings.TrimSpace(*in.UnitLabel)
		if utf8.RuneCountInString(label) > maxLabelRunes {
			fields["unit_label"] = fmt.Sprintf("Keep it under %d characters.", maxLabelRunes)
		} else {
			g.UnitLabel = label
		}
	}

	g.StartDate = optionalDate(in.StartDate, g.StartDate, creating, "start_date", fields)
	g.EndDate = optionalDate(in.EndDate, g.EndDate, creating, "end_date", fields)

	// A goal that ends before it starts is the one pair of valid dates that
	// cannot both be true.
	if !g.StartDate.IsZero() && !g.EndDate.IsZero() && g.EndDate.Before(g.StartDate) {
		fields["end_date"] = "This is before it starts."
	}

	if in.Done != nil {
		g.Done = *in.Done
	}

	if len(fields) > 0 {
		return Goal{}, apperr.Validation(fields)
	}
	return g, nil
}

func optionalDate(in *string, current time.Time, required bool, field string, fields map[string]string) time.Time {
	if in == nil {
		if required && current.IsZero() {
			fields[field] = "Pick a date."
		}
		return current
	}
	parsed, err := time.Parse(time.DateOnly, strings.TrimSpace(*in))
	if err != nil {
		fields[field] = "Pick a date."
		return current
	}
	return parsed
}

// ValidateAmount refuses zero (records nothing); negative is allowed as a correction, so the entry isn't just deleted.
func ValidateAmount(amount int64) error {
	if amount == 0 {
		return apperr.Validation(map[string]string{"amount": "Enter an amount."})
	}
	return nil
}
