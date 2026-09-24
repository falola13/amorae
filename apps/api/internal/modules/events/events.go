// Package events owns the things a couple plans together.
//
// There is no owner and no participants list: both partners are implicit
// participants in everything (DEC-16). So every rule here is about the shape
// of an event, never about who may touch it — the answer to that is always
// "either of them", and the answer to "whose event is this" is always the
// couple's (DEC-19).
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
	maxTitleRunes    = 80
	maxLocationRunes = 120
	maxReminderRunes = 40
	maxNotesRunes    = 1000
	maxItemRunes     = 120
	// A checklist longer than this is a different feature, and a screen that
	// has to scroll to tick something off is not helping anyone.
	MaxChecklistItems = 20
)

var ErrNotFound = apperr.NotFound("event_not_found", "That event isn’t here.")

var (
	day       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	clockTime = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

// Event is one thing the two of them are doing.
type Event struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	Title     string
	Date      time.Time
	StartTime string // "19:00", or empty
	EndTime   string
	Location  string
	Reminder  string
	Notes     string
	Done      bool
	Checklist []ChecklistItem
}

type ChecklistItem struct {
	ID       uuid.UUID
	Position int
	Text     string
	Done     bool
}

// Input is a create or an edit. Every field is a pointer so an edit can send
// one of them: nil means "leave it", which is not the same as "clear it".
type Input struct {
	Title     *string
	Date      *string
	StartTime *string
	EndTime   *string
	Location  *string
	Reminder  *string
	Notes     *string
	Checklist *[]string
}

// Validate applies an input to an event and reports everything wrong with it
// at once, so nobody fixes one field only to be told about the next.
//
// `creating` decides whether the required fields have to be present: an edit
// that does not mention the title is leaving it alone, while a create that
// does not mention it has no title at all.
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
	e.Reminder = optionalText(in.Reminder, e.Reminder, maxReminderRunes, "reminder", fields)
	e.Notes = optionalText(in.Notes, e.Notes, maxNotesRunes, "notes", fields)

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

// optionalTime keeps "" meaning "there isn't one", so clearing a start time
// is sending an empty string rather than a separate kind of request.
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

// validateChecklist takes the whole list, so the order given is the order
// kept and an item dropped from it is an item deleted.
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
			// Blank rows come from an empty input somebody never filled in.
			// Dropping them is kinder than making them fix it.
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
