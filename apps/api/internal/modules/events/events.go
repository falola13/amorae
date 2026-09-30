// Package events owns the things a couple plans together. No owner or
// participants list — both partners are implicit in everything (DEC-16, DEC-19).
package events

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

const (
	maxTitleRunes     = 80
	maxLocationRunes  = 120
	MaxReminders      = 3
	maxNotesRunes     = 1000
	maxItemRunes      = 120
	MaxChecklistItems = 20
)

var ErrNotFound = apperr.NotFound("event_not_found", "That event isn’t here.")

var (
	day       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	clockTime = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

	// The two shapes of reminder the worker reads besides the fixed phrases:
	// a lead ("an hour before", "45 minutes before") and a set clock time on
	// the event's day. Kept in step with notifications.EventReminderAt, which
	// is what turns them into a send time — events cannot import it.
	reminderLead = regexp.MustCompile(`^(\d{1,3}|a|an|the) (minute|hour|day)s? before$`)
	reminderAt   = regexp.MustCompile(`^at ([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

type Event struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	// Who made it. Nil on events from before ownership existed, and on
	// nothing created since (the app always sets it) — treated the same as
	// "together" wherever it matters.
	CreatedBy *uuid.UUID
	// "together" (both partners see it and either may edit it, as before) or
	// "mine" (its creator only — reads still reach both, DEC-16 is about
	// seeing, not about touching).
	Kind      string
	Title     string
	Date      time.Time
	StartTime string // "19:00", or empty
	EndTime   string
	Location  string
	// In the order they were given; empty means none.
	Reminders []string
	Notes     string
	// Done and DidntHappen are the outcome, never both true; neither is "not
	// said yet".
	Done        bool
	DidntHappen bool
	Checklist   []ChecklistItem
}

const (
	KindTogether = "together"
	KindMine     = "mine"
)

// What can be said about how an event went; "none" takes it back.
const (
	OutcomeHappened    = "happened"
	OutcomeDidntHappen = "didnt_happen"
	OutcomeNone        = "none"
)

type ChecklistItem struct {
	ID       uuid.UUID
	Position int
	Text     string
	Done     bool
}

// Input is a create or edit; every field is a pointer — nil means "leave it", not the same as "clear it".
type Input struct {
	Title     *string
	Date      *string
	StartTime *string
	EndTime   *string
	Location  *string
	Reminders *[]string
	Notes     *string
	Checklist *[]string
	Kind      *string
}

// Validate reports every invalid field at once. creating gates required
// fields: an edit omitting a field leaves it alone, a create omitting it has none.
func (e Event) Validate(in Input, creating bool) (Event, error) {
	fields := map[string]string{}

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		switch {
		case title == "":
			fields["title"] = "What would you like to do together?"
		case utf8.RuneCountInString(title) > maxTitleRunes:
			fields["title"] = fmt.Sprintf("Keep it under %d characters.", maxTitleRunes)
		default:
			e.Title = title
		}
	} else if creating {
		fields["title"] = "What would you like to do together?"
	}

	if in.Date != nil {
		date := strings.TrimSpace(*in.Date)
		parsed, err := time.Parse(time.DateOnly, date)
		if !day.MatchString(date) || err != nil {
			fields["date"] = "Pick a date."
		} else {
			e.Date = parsed
		}
	} else if creating {
		fields["date"] = "Pick a date."
	}

	e.StartTime = optionalTime(in.StartTime, e.StartTime, "start_time", fields)
	e.EndTime = optionalTime(in.EndTime, e.EndTime, "end_time", fields)

	// An end before a start is the one combination of two valid times that
	// cannot be true.
	if e.StartTime != "" && e.EndTime != "" && e.EndTime < e.StartTime {
		fields["end_time"] = "This is before it starts."
	}

	e.Location = optionalText(in.Location, e.Location, maxLocationRunes, "location", fields)
	if in.Reminders != nil {
		reminders, msg := validateReminders(*in.Reminders)
		if msg != "" {
			fields["reminders"] = msg
		} else {
			e.Reminders = reminders
		}
	}
	e.Notes = optionalText(in.Notes, e.Notes, maxNotesRunes, "notes", fields)

	if in.Kind != nil {
		switch strings.TrimSpace(*in.Kind) {
		case KindTogether, KindMine:
			e.Kind = strings.TrimSpace(*in.Kind)
		default:
			fields["kind"] = "Choose together or just you."
		}
	} else if creating && e.Kind == "" {
		// Nobody has to say "together" on the way in; it is what an event
		// has always been.
		e.Kind = KindTogether
	}

	if in.Checklist != nil {
		items, err := validateChecklist(*in.Checklist)
		if err != nil {
			if appErr, ok := err.(*apperr.Error); ok {
				for k, v := range appErr.Fields {
					fields[k] = v
				}
			}
		} else {
			e.Checklist = items
		}
	}

	if len(fields) > 0 {
		return Event{}, apperr.Validation(fields)
	}
	return e, nil
}

// optionalTime treats "" as "there isn't one"; clearing sends an empty string, not a separate request kind.
func optionalTime(in *string, current, field string, fields map[string]string) string {
	if in == nil {
		return current
	}
	value := strings.TrimSpace(*in)
	if value == "" {
		return ""
	}
	if !clockTime.MatchString(value) {
		fields[field] = "Use a time like 19:00."
		return current
	}
	return value
}

func optionalText(in *string, current string, max int, field string, fields map[string]string) string {
	if in == nil {
		return current
	}
	value := strings.TrimSpace(*in)
	if utf8.RuneCountInString(value) > max {
		fields[field] = fmt.Sprintf("Keep it under %d characters.", max)
		return current
	}
	return value
}

// validateChecklist replaces the whole list: order is as given, and a dropped item is deleted.
func validateChecklist(texts []string) ([]ChecklistItem, error) {
	if len(texts) > MaxChecklistItems {
		return nil, apperr.Validation(map[string]string{
			"checklist": fmt.Sprintf("Keep it to %d things or fewer.", MaxChecklistItems),
		})
	}

	fields := map[string]string{}
	items := make([]ChecklistItem, 0, len(texts))
	for i, text := range texts {
		text = strings.TrimSpace(text)
		switch {
		case text == "":
			// Blank rows are dropped rather than rejected.
			continue
		case utf8.RuneCountInString(text) > maxItemRunes:
			fields[fmt.Sprintf("checklist.%d", i)] = fmt.Sprintf("Keep it under %d characters.", maxItemRunes)
		default:
			items = append(items, ChecklistItem{Position: len(items), Text: text})
		}
	}
	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	return items, nil
}

// validateReminders replaces the whole list, like the checklist: the order is
// kept, and blank entries are dropped rather than rejected. Phrases are read
// case-insensitively and stored lowercased, so "1 Hour Before" and
// "1 hour before" are the same reminder.
func validateReminders(in []string) ([]string, string) {
	const choices = "Up to three reminders, each one of the choices."
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, r := range in {
		r = strings.ToLower(strings.TrimSpace(r))
		switch {
		case r == "":
			continue
		case !validReminder(r):
			return nil, choices
		case seen[r]:
			return nil, "Pick each reminder only once."
		}
		seen[r] = true
		out = append(out, r)
	}
	if len(out) > MaxReminders {
		return nil, choices
	}
	return out, ""
}

func validReminder(r string) bool {
	switch r {
	case "at the time", "the morning of":
		return true
	}
	if reminderAt.MatchString(r) {
		return true
	}
	m := reminderLead.FindStringSubmatch(r)
	if m == nil {
		return false
	}
	// "0 hours before" is not a lead.
	return strings.Trim(m[1], "0") != ""
}
