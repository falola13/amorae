// Package anniversary owns the couple's anniversary as a day of its own:
// which one is next, the letters they write each other for it, and the year
// that led up to it.
//
// The one rule that matters here is the seal. A letter is written in the
// days before and its text is never sent to the other partner until the
// couple's own calendar reaches the day — not hidden by the client, withheld
// by the server.
package anniversary

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// WriteAhead is how many days before the day a letter can be started. Two
// weeks: enough to come back to it, not so long that it is a chore hanging
// over them.
const WriteAhead = 14

const maxLetterRunes = 6000

var (
	// Not a failure: a couple without a start date, or less than a year in,
	// has no anniversary yet. The client treats it as "nothing to show".
	ErrNoAnniversary = apperr.NotFound("no_anniversary", "There’s no anniversary to mark yet.")
	ErrTooEarly      = apperr.Conflict("too_early",
		fmt.Sprintf("Letters open for writing %d days before the day.", WriteAhead))
)

// Occasion is the anniversary being looked forward to, or today's.
type Occasion struct {
	// The day, as a calendar date (midnight UTC, the convention for DATE here).
	On time.Time
	// How many years it marks; always at least one.
	Years int
	// Days from today; 0 on the day itself.
	DaysUntil int
}

// Writable reports whether a letter for this occasion can be written or changed now.
func (o Occasion) Writable() bool { return o.DaysUntil <= WriteAhead }

// YearBefore is the start of the year this occasion closes: the same day,
// one year earlier (Feb 29 stepping back to Feb 28).
func (o Occasion) YearBefore() time.Time {
	return dayIn(o.On.Year()-1, o.On.Month(), o.On.Day())
}

// Next is the anniversary on or after today, given when they started. False
// when there isn't one worth marking: no start date, or not a year in yet.
// Feb 29 falls on Feb 28 in a short year, the same as notifications.OccursOn.
func Next(start *time.Time, today time.Time) (Occasion, bool) {
	if start == nil {
		return Occasion{}, false
	}
	today = calendar(today)
	on := dayIn(today.Year(), start.Month(), start.Day())
	if on.Before(today) {
		on = dayIn(today.Year()+1, start.Month(), start.Day())
	}
	years := on.Year() - start.Year()
	if years < 1 {
		return Occasion{}, false
	}
	return Occasion{On: on, Years: years, DaysUntil: int(on.Sub(today).Hours() / 24)}, true
}

// LocalToday is the couple's calendar date at `now`, in their own zone.
func LocalToday(now time.Time, timezone string) time.Time {
	zone, err := time.LoadLocation(timezone)
	if err != nil {
		zone = time.UTC
	}
	return calendar(now.In(zone))
}

func calendar(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func dayIn(year int, month time.Month, day int) time.Time {
	if month == time.February && day == 29 && !isLeapYear(year) {
		day = 28
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func isLeapYear(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

// Letter is one person's letter for one anniversary.
type Letter struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	AuthorID  uuid.UUID
	OpensOn   time.Time
	Text      string
	UpdatedAt time.Time
}

// Open reports whether the letter may be read by the person it is for.
func (l Letter) Open(today time.Time) bool { return !l.OpensOn.After(calendar(today)) }

// ValidateLetter trims and bounds a letter.
func ValidateLetter(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		return "", apperr.Validation(map[string]string{"text": "Write a little before you keep it."})
	case utf8.RuneCountInString(text) > maxLetterRunes:
		return "", apperr.Validation(map[string]string{
			"text": fmt.Sprintf("Keep it under %d characters.", maxLetterRunes),
		})
	}
	return text, nil
}

// Year is what the couple did in the year an anniversary closes.
type Year struct {
	From, To time.Time
	// Each time either of them marked a prayer prayed.
	Prayed int
	// Prayers marked answered.
	Answered int
	// Weeks they shared prayer.
	PrayerWeeks int
	Memories    int
	// Plans that happened (not the ones said not to have).
	Events     int
	Challenges int
	Notes      int
	Journal    int
}
