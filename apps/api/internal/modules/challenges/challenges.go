// Package challenges owns short, guided, multi-day experiences a couple
// takes on together. Each partner marks their own days (DEC-30); skipping is
// a first-class answer, not a penalty (01 §6).
package challenges

import (
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var (
	ErrNotFound = apperr.NotFound("challenge_not_found", "You don’t have a challenge going.")
	// Only one challenge at a time.
	ErrAlreadyRunning = apperr.Conflict("challenge_already_running",
		"You already have a challenge going. Finish or leave that one first.")
	ErrUnknownTemplate = apperr.Invalid("challenge_unknown", "That challenge isn’t one of ours.")
	ErrUnknownDay      = apperr.NotFound("challenge_day_not_found", "That day isn’t part of this challenge.")
)

// Mark is what one partner has said about one day.
type Mark string

const (
	MarkDone    Mark = "done"
	MarkSkipped Mark = "skipped"
)

type Challenge struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	Template  string
	Title     string
	StartedOn time.Time
	Days      []Day
}

type Day struct {
	ID     uuid.UUID
	N      int
	Prompt string
	// Absent means unmarked, distinct from skipped.
	Marks map[uuid.UUID]Mark
}

func (d Day) MarkFor(userID uuid.UUID) (Mark, bool) {
	m, ok := d.Marks[userID]
	return m, ok
}

// Template is a curated challenge. Kept in code, not a DB table — these read
// like copy and a typo fix shouldn't need a migration.
type Template struct {
	Key     string
	Title   string
	Blurb   string
	Prompts []string
}

var templates = []Template{
	{
		Key:   "seven-days-of-noticing",
		Title: "Seven days of noticing",
		Blurb: "One small thing a day, about each other.",
		Prompts: []string{
			"Tell them one thing they did this week that you noticed.",
			"Ask about something they are looking forward to.",
			"Say thank you for something ordinary.",
			"Put your phones down for one meal together.",
			"Tell them something you admire that they would not guess.",
			"Ask what would make tomorrow easier for them, and do it.",
			"Say what this week has been like for you, honestly.",
		},
	},
	{
		Key:   "seven-days-of-praying-together",
		Title: "Seven days of praying together",
		Blurb: "A short prayer each day, out loud, together.",
		Prompts: []string{
			"Pray for each other's week ahead.",
			"Pray for your families.",
			"Pray about something one of you is worried about.",
			"Give thanks for three things from this year.",
			"Pray for someone outside the two of you.",
			"Pray about your home, wherever it is.",
			"Pray for the two of you, five years from now.",
		},
	},
	{
		Key:   "fourteen-days-of-small-things",
		Title: "Fourteen days of small things",
		Blurb: "Two weeks of very small, very doable things.",
		Prompts: []string{
			"Make them a drink the way they like it.",
			"Send a message in the middle of the day for no reason.",
			"Take one chore off their list without mentioning it.",
			"Ask about their day and let them finish.",
			"Share a memory from before you lived together.",
			"Go to bed at the same time.",
			"Tell them one thing you are proud of them for.",
			"Plan something small for next weekend.",
			"Sit outside together for ten minutes.",
			"Ask what they need this week.",
			"Say sorry for something small you never said sorry for.",
			"Look at old photos together.",
			"Do nothing together, on purpose, for half an hour.",
			"Tell them what these two weeks have been like.",
		},
	},
}

func Templates() []Template { return templates }

func TemplateByKey(key string) (Template, error) {
	for _, t := range templates {
		if t.Key == key {
			return t, nil
		}
	}
	return Template{}, ErrUnknownTemplate
}

// ValidateMark requires done or skipped; explicitly false means un-marking.
func ValidateMark(done, skipped *bool) (Mark, bool, error) {
	switch {
	case done != nil && *done:
		return MarkDone, true, nil
	case skipped != nil && *skipped:
		return MarkSkipped, true, nil
	case done != nil || skipped != nil:
		// Explicitly false: they are taking it back.
		return "", false, nil
	default:
		return "", false, apperr.Validation(map[string]string{
			"done": "Say whether it is done or skipped.",
		})
	}
}
