// Package prayers owns the weekly prayer cycle: turn rotation, valid prayer
// points, and per-partner visibility. Pure domain logic only — no SQL, no
// HTTP, no clock — so it's testable without a database.
package prayers

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// MaxPoints is a deliberate kindness, not a technical bound.
const MaxPoints = 10

// AllWeekdays is the weekday bitmask meaning "every day" — the default for
// a point that says nothing about which days it's for. Bit n is weekday n
// (Sunday=0 .. Saturday=6, matching time.Weekday).
const AllWeekdays = 127

const (
	maxTitleRunes      = 80
	maxBodyRunes       = 500
	maxScriptureRunes  = 60
	maxVerseRunes      = 500
	maxReflectionRunes = 2000
	maxAnswerRunes     = 1000
)

type Status string

const (
	// Not yet shared, but visible to both partners — either may write it
	// (DEC-33).
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

var (
	ErrNotFound = apperr.NotFound("prayer_week_not_found", "That prayer week isn’t here.")
	ErrNotDraft = apperr.Conflict("week_already_published", "This week has been shared already.")
	// A draft prayer hasn't been seen by the partner, so it can't be answered.
	ErrNotShared = apperr.Conflict("prayer_not_shared", "This one hasn’t been shared yet.")
	// No week exists until the couple has two members to set a rotation.
	ErrWaitingForPartner = apperr.Conflict("waiting_for_partner", "Your first prayer week starts when your partner joins.")
	// A point not scheduled for today, or a week that isn't the current one —
	// a couple prays the week's points every day, but only today's points,
	// and only for the week that is actually running now.
	ErrNotForToday = apperr.Conflict("not_for_today", "This one isn’t for today.")
)

// Week is one couple's week of prayer.
type Week struct {
	ID           uuid.UUID
	CoupleID     uuid.UUID
	WeekStart    time.Time // the couple's local Sunday, date only
	SetterUserID uuid.UUID
	Status       Status
	PublishedAt  *time.Time
	// Who published it — nil for a week from before either partner could,
	// or one still in draft. Decides who KindWeekPublished is addressed to.
	PublishedBy *uuid.UUID
	Points      []Point
}

// Point finds one of the week's own points by id.
func (w Week) Point(id uuid.UUID) (Point, bool) {
	for _, p := range w.Points {
		if p.ID == id {
			return p, true
		}
	}
	return Point{}, false
}

// Point is one thing to pray about. Body, scripture and verse are optional —
// a title alone is a complete prayer point.
type Point struct {
	ID        uuid.UUID
	Position  int
	Title     string
	Body      string
	Scripture string
	Verse     string

	// Weekdays is a bitmask, bit n meaning "scheduled on weekday n" (Sunday=0
	// .. Saturday=6). Always in [1, AllWeekdays] once validated — never zero,
	// which would mean "never". AllWeekdays is every day, the default.
	Weekdays int
	// WeekdaysRaw is what a request carries before ValidatePoints turns it
	// into Weekdays; empty means every day. Never read afterwards, and never
	// persisted itself.
	WeekdaysRaw []int

	// Answered once per couple, not per person, so a single nullable time
	// rather than a join table. AnsweredBy is who noticed.
	AnsweredAt *time.Time
	// AnsweredAt as a date in the couple's own timezone (computed in SQL),
	// so it reads as the same day to both partners regardless of server UTC.
	AnsweredOn *time.Time
	AnsweredBy uuid.UUID
	AnswerNote string
}

// Answered carries enough of its week to place it in time, since it's shown
// across every week a couple has had.
type Answered struct {
	Point
	WeekID    uuid.UUID
	WeekStart time.Time
}

// Member is who, and when they joined — join order fixes the rotation.
type Member struct {
	UserID   uuid.UUID
	JoinedAt time.Time
}

// StartOfWeek is the Sunday 00:00 that `at` falls in, in the couple's own
// timezone (not the server's, not each partner's — they must agree on the week).
func StartOfWeek(at time.Time, loc *time.Location) time.Time {
	local := at.In(loc)
	daysSinceSunday := int(local.Weekday()) // time.Sunday is 0
	y, m, d := local.AddDate(0, 0, -daysSinceSunday).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// WeekIndex counts whole weeks between two StartOfWeek values; both are
// date-only so daylight saving can't skew the count.
func WeekIndex(firstWeekStart, weekStart time.Time) int {
	return int(weekStart.Sub(firstWeekStart).Hours() / (24 * 7))
}

// StartOfDay is the couple-local calendar date `at` falls on, in the same
// date-only representation as StartOfWeek (midnight UTC standing in for a
// date, never an instant) — "today" is a calendar fact, not a moment.
func StartOfDay(at time.Time, loc *time.Location) time.Time {
	local := at.In(loc)
	y, m, d := local.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ScheduledOn reports whether a point carrying this weekday mask is due on
// `day` — the couple-local weekday of a date-only time.Time from StartOfDay
// or StartOfWeek.AddDate.
func ScheduledOn(weekdays int, day time.Weekday) bool {
	return weekdays&(1<<uint(day)) != 0
}

// SetterFor is whose turn it is: members in join order, alternating weekly.
// Stored on creation and never recomputed, so the turn can't move retroactively (DEC-18).
func SetterFor(members []Member, weekIndex int) (uuid.UUID, error) {
	if len(members) != 2 {
		return uuid.UUID{}, fmt.Errorf("a prayer week needs exactly two members, got %d", len(members))
	}
	ordered := [2]Member{members[0], members[1]}
	if ordered[1].JoinedAt.Before(ordered[0].JoinedAt) {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	// Go's % keeps the sign of the dividend; normalise into {0,1}.
	return ordered[((weekIndex%2)+2)%2].UserID, nil
}

// ValidatePoints reports every problem at once, and assigns Position from
// the order given.
func ValidatePoints(points []Point) ([]Point, error) {
	if len(points) > MaxPoints {
		return nil, apperr.Validation(map[string]string{
			"points": fmt.Sprintf("Keep it to %d prayers or fewer.", MaxPoints),
		})
	}

	fields := map[string]string{}
	cleaned := make([]Point, 0, len(points))
	for i, p := range points {
		p.Position = i
		p.Title = strings.TrimSpace(p.Title)
		p.Body = strings.TrimSpace(p.Body)
		p.Scripture = strings.TrimSpace(p.Scripture)
		p.Verse = strings.TrimSpace(p.Verse)

		switch {
		case p.Title == "":
			fields[fmt.Sprintf("points.%d.title", i)] = "Give this prayer a title."
		case utf8.RuneCountInString(p.Title) > maxTitleRunes:
			fields[fmt.Sprintf("points.%d.title", i)] = fmt.Sprintf("Keep the title under %d characters.", maxTitleRunes)
		}
		if utf8.RuneCountInString(p.Body) > maxBodyRunes {
			fields[fmt.Sprintf("points.%d.body", i)] = fmt.Sprintf("Keep it under %d characters.", maxBodyRunes)
		}
		if utf8.RuneCountInString(p.Scripture) > maxScriptureRunes {
			fields[fmt.Sprintf("points.%d.scripture", i)] = fmt.Sprintf("Keep the reference under %d characters.", maxScriptureRunes)
		}
		if utf8.RuneCountInString(p.Verse) > maxVerseRunes {
			fields[fmt.Sprintf("points.%d.verse", i)] = fmt.Sprintf("Keep the verse under %d characters.", maxVerseRunes)
		}

		// Empty means every day; a client never has to spell that out.
		mask := AllWeekdays
		if len(p.WeekdaysRaw) > 0 {
			mask = 0
			for _, day := range p.WeekdaysRaw {
				if day < 0 || day > 6 {
					fields[fmt.Sprintf("points.%d.weekdays", i)] = "Days must be between Sunday (0) and Saturday (6)."
					mask = AllWeekdays
					break
				}
				mask |= 1 << uint(day)
			}
		}
		p.Weekdays = mask
		p.WeekdaysRaw = nil

		cleaned = append(cleaned, p)
	}

	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	return cleaned, nil
}

// ValidateReflection bounds one partner's note on the week.
func ValidateReflection(body string) (string, error) {
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > maxReflectionRunes {
		return "", apperr.Validation(map[string]string{
			"reflection": fmt.Sprintf("Keep it under %d characters.", maxReflectionRunes),
		})
	}
	return body, nil
}

// ValidateAnswerNote bounds the line about what happened. Empty is allowed —
// marking a prayer answered shouldn't require writing something.
func ValidateAnswerNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxAnswerRunes {
		return "", apperr.Validation(map[string]string{
			"note": fmt.Sprintf("Keep it under %d characters.", maxAnswerRunes),
		})
	}
	return note, nil
}

// CanAnswer says whether this point may be marked answered. Either partner
// may; the only bar is that the week has actually been shared.
func CanAnswer(w Week) error {
	if w.Status != StatusPublished {
		return ErrNotShared
	}
	return nil
}

// StatusFor is what `viewer` should be told the week's status is. Both
// partners are told the same thing, draft or published — either may edit or
// publish a week at any time (DEC-33), so hiding it from one of them would
// just be hiding work they're allowed to do. Kept as a function rather than
// a raw field read so this boundary stays in one place.
func StatusFor(w Week, viewer uuid.UUID) Status {
	return w.Status
}

// PointsFor is the week's points as `viewer` may see them: the same points
// either partner sees, for the reason StatusFor gives.
func PointsFor(w Week, viewer uuid.UUID) []Point {
	return w.Points
}

// Publishing is open to either partner (DEC-33); a caller has already been
// checked to belong to the couple by the time it reaches here, so there is
// nothing left for a CanPublish to guard — the service just publishes.
