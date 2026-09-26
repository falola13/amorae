// Package milestones owns the dates a couple keeps: birthdays, anniversaries,
// the day they met. One entity, not one per kind (BR-DATE-01). A row is the
// day it happened; "when is this next" is worked out by callers, not stored.
package milestones

import (
	"crypto/md5"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var (
	ErrNotFound = apperr.NotFound("not_found", "That date isn’t here.")
	// ErrDerived is Delete's answer for an id that isn't a row here at
	// all — it's the anniversary or a birthday, worked out from the couple
	// or a profile, and there is nothing here to remove.
	ErrDerived = apperr.Conflict("milestone_derived", "This one comes from your space or a profile — change it there.")
)

const (
	maxTitleRunes = 80
	maxSubRunes   = 80
)

// Source says a Milestone was computed from another record rather than
// stored here. Empty for a stored one.
type Source string

const (
	SourceAnniversary Source = "anniversary"
	SourceBirthday    Source = "birthday"
)

type Milestone struct {
	ID       uuid.UUID
	CoupleID uuid.UUID
	Title    string
	Date     time.Time
	Sub      string
	// Whether this one comes round for them every year, or is simply kept.
	Reminder bool
	// Source is set for a date computed from the couple itself or a
	// profile (see Derived), and empty for one actually stored here.
	Source Source
	// YearKnown only means anything for a SourceBirthday: false when that
	// profile has no birth year, in which case Date's year is the 2000
	// placeholder ValidateBirthday uses, not a real one.
	YearKnown bool
	// About is who a SourceBirthday date belongs to; the zero uuid for
	// anything else.
	About uuid.UUID
}

// DerivedID is the id given to a computed date: deterministic from the seed
// that names it, so the same couple (and the same member, for a birthday)
// always gets the same id back. Matched, not shared, with the identical
// expression on the SQL side — notifications' worker_repository.go
// ImportantDates query — because that query has to produce these rows
// without a round trip through Go; TestDerivedID_MatchesSQL keeps the two
// from drifting apart.
//
// md5(seed)::uuid in Postgres reinterprets the raw digest as a uuid without
// touching version/variant bits, so this does the same: cast, not hash of a
// UUID namespace (which uuid.NewMD5 would give a different answer for).
func DerivedID(seed string) uuid.UUID {
	return uuid.UUID(md5.Sum([]byte(seed)))
}

// AnniversaryID and BirthdayID build DerivedID's seed consistently, so the
// exact format only has to be remembered in one place.
func AnniversaryID(coupleID uuid.UUID) uuid.UUID {
	return DerivedID(coupleID.String() + ":anniversary")
}

func BirthdayID(coupleID, userID uuid.UUID) uuid.UUID {
	return DerivedID(coupleID.String() + ":birthday:" + userID.String())
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
