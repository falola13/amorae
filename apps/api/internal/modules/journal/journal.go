// Package journal owns the couple's shared journal: short entries, each
// tagged, each with an author.
//
// The author is the difference between this and memories. A memory is
// something that happened to them both; a journal entry is something one of
// them wrote, and reading it back years later is partly about which of them
// was thinking it.
package journal

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

const maxTextRunes = 2000

// Tag is what an entry is: the five the composer offers, spelled the way it
// sends them.
type Tag string

const (
	TagGratitude    Tag = "Gratitude"
	TagReflection   Tag = "Reflection"
	TagMemory       Tag = "Memory"
	TagAppreciation Tag = "Appreciation"
	TagPlans        Tag = "Plans"
)

var tags = []Tag{TagGratitude, TagReflection, TagMemory, TagAppreciation, TagPlans}

// Entry is one thing one of them wrote.
type Entry struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	AuthorID uuid.UUID
	Date     time.Time
	Tag      Tag
	Text     string
}

// Validate checks a new entry and returns it cleaned.
//
// The tag is matched exactly rather than case-insensitively: it is a value
// from a picker, not something anybody types, so a different spelling means a
// client sending something this server has never offered.
func Validate(tag, text string) (Tag, string, error) {
	text = strings.TrimSpace(text)

	fields := map[string]string{}
	if !valid(Tag(tag)) {
		fields["tag"] = fmt.Sprintf("Choose one of: %s.", list())
	}
	switch {
	case text == "":
		fields["text"] = "Write a line first."
	case utf8.RuneCountInString(text) > maxTextRunes:
		fields["text"] = fmt.Sprintf("Keep it under %d characters.", maxTextRunes)
	}
	if len(fields) > 0 {
		return "", "", apperr.Validation(fields)
	}
	return Tag(tag), text, nil
}

func valid(t Tag) bool {
	for _, known := range tags {
		if known == t {
			return true
		}
	}
	return false
}

func list() string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, string(t))
	}
	return strings.Join(out, ", ")
}
