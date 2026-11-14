// Package questions owns the daily question: one for the couple each day,
// answered by each of them in private. The seal is kept here, on the server:
// a partner's answer is only returned to someone who has answered that same
// day themselves, so no client can show what it was never sent.
package questions

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

const maxAnswerRunes = 1000

// Answer is one person's answer to one day's question.
type Answer struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	AuthorID  uuid.UUID
	AskedOn   time.Time // a calendar day, midnight UTC
	Question  string    // as asked that day
	Text      string
	UpdatedAt time.Time
}

// Entry is one day's question as one person sees it.
type Entry struct {
	AskedOn  time.Time
	Question string
	Mine     *Answer
	// Theirs is only ever set once Mine is; TheirsAnswered says they have
	// answered without saying what.
	Theirs         *Answer
	TheirsAnswered bool
	// Previous is the last time this same question was asked and both of you
	// answered it, for Today only. It is sealed by the same rule as any day.
	Previous *Entry
}

// ValidateAnswer trims and bounds an answer.
func ValidateAnswer(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		return "", apperr.Validation(map[string]string{"text": "Write a little before you send it."})
	case utf8.RuneCountInString(text) > maxAnswerRunes:
		return "", apperr.Validation(map[string]string{
			"text": fmt.Sprintf("Keep it under %d characters.", maxAnswerRunes),
		})
	}
	return text, nil
}

// LocalToday is the couple's calendar date at now, in their own zone.
func LocalToday(now time.Time, timezone string) time.Time {
	zone, err := time.LoadLocation(timezone)
	if err != nil {
		zone = time.UTC
	}
	t := now.In(zone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// assemble is where the seal is kept, for Today and Past alike: the
// partner's text is placed in Theirs only when the caller answered that day.
// asked is the question to show when nobody has answered yet.
func assemble(userID uuid.UUID, day time.Time, asked string, answers []Answer) Entry {
	e := Entry{AskedOn: day, Question: asked}
	var theirs *Answer
	for i := range answers {
		a := answers[i]
		e.Question = a.Question
		if a.AuthorID == userID {
			e.Mine = &a
		} else {
			theirs = &a
		}
	}
	if theirs != nil {
		e.TheirsAnswered = true
		if e.Mine != nil {
			e.Theirs = theirs
		}
	}
	return e
}
