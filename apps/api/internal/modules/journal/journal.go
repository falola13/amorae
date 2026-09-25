// Package journal owns the couple's shared journal: short, tagged entries,
// each with an author — the difference from memories, which have none.
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

// Tag values are spelled exactly as the composer sends them.
type Tag string

const (
	TagGratitude    Tag = "Gratitude"
	TagReflection   Tag = "Reflection"
	TagMemory       Tag = "Memory"
	TagAppreciation Tag = "Appreciation"
	TagPlans        Tag = "Plans"
)

var tags = []Tag{TagGratitude, TagReflection, TagMemory, TagAppreciation, TagPlans}

type Entry struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	AuthorID uuid.UUID
	Date     time.Time
	Tag      Tag
	Text     string
}

// Validate matches the tag exactly, not case-insensitively — it's from a picker, not free text.
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
