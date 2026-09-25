// Package memories owns the moments a couple keeps: a title, the day, and
// optionally where it was and a note. Deliberately minimal — no author,
// reactions, or comments; it's an archive, not a feed.
package memories

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var (
	ErrNotFound = apperr.NotFound("not_found", "That moment isn’t here.")
	// Distinct from a generic failure: photos need an account elsewhere that may not be configured.
	ErrNoPhotos = apperr.Invalid("photos_unavailable", "Photos aren’t set up on this server yet.")
)

const (
	maxTitleRunes    = 80
	maxLocationRunes = 80
	maxNoteRunes     = 500
)

// Memory is one kept moment.
type Memory struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	Title    string
	Date     time.Time
	Location string
	Note     string
	// Cloudinary public id; empty means no photo. No separate "has a photo"
	// flag to fall out of step with it (FR-MEM-003).
	PhotoID string
	// Versions the photo's delivery URL (Service.PhotoURL).
	UpdatedAt time.Time
}

func (m Memory) HasPhoto() bool { return m.PhotoID != "" }

type Input struct {
	Title    string
	Date     time.Time
	Location string
	Note     string
}

// Validate trims and bounds fields using the same wording as the composer, so errors are recognizable.
func Validate(in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Location = strings.TrimSpace(in.Location)
	in.Note = strings.TrimSpace(in.Note)

	fields := map[string]string{}
	switch {
	case in.Title == "":
		fields["title"] = "What happened?"
	case utf8.RuneCountInString(in.Title) > maxTitleRunes:
		fields["title"] = "Keep it short."
	}
	if utf8.RuneCountInString(in.Location) > maxLocationRunes {
		fields["location"] = "Keep it short."
	}
	if utf8.RuneCountInString(in.Note) > maxNoteRunes {
		fields["note"] = "Keep it under 500 characters."
	}
	if in.Date.IsZero() {
		fields["date"] = "Pick a date."
	}
	if len(fields) > 0 {
		return Input{}, apperr.Validation(fields)
	}
	return in, nil
}
