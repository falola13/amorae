// Package memories owns the moments a couple keeps: a title, the day, and
// optionally where it was and a line about it.
//
// It is deliberately the smallest module in the app. A memory has no author,
// no reactions and no comments — the point is a shared archive to read back,
// not a feed to perform in, and every feature that would make it a feed is
// one the two of them would then have to manage.
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
	// Photos need an account somewhere else, so this says so plainly rather
	// than failing as though something broke.
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
	// Where the photo is, if there is one — a Cloudinary public id the server
	// derived. Empty means no photo. There is no separate "has a photo" flag
	// to fall out of step with it (FR-MEM-003).
	PhotoID string
	// Versions the photo's delivery URL (Service.PhotoURL).
	UpdatedAt time.Time
}

// HasPhoto is the question screens actually ask.
func (m Memory) HasPhoto() bool { return m.PhotoID != "" }

// Input is what either partner sends to keep one.
type Input struct {
	Title    string
	Date     time.Time
	Location string
	Note     string
}

// Validate trims and bounds an input, in the same words the composer uses so
// a person who sees an error recognises the field it belongs to.
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
