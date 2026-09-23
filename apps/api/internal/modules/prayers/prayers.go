// Package prayers owns the weekly prayer cycle: whose turn it is to set the
// week, what a valid set of prayer points looks like, and what each partner
// may see and change.
//
// This file is the rules, and nothing else: no SQL, no HTTP, no clock of its
// own. Everything here is a pure function of its arguments, which is why it
// can be tested without a database and why the service above it has only one
// job — deciding when to apply these rules.
package prayers

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// A week holds at most this many points. The limit is a kindness, not a
// technical bound: ten things to pray about is already a lot to hold.
const MaxPoints = 10

const (
	maxTitleRunes      = 80
	maxBodyRunes       = 500
	maxScriptureRunes  = 60
	maxVerseRunes      = 500
	maxReflectionRunes = 2000
)

type Status string

const (
	// The setter is still writing; only they can see the points.
	StatusDraft Status = "draft"
	// Shared with both partners.
	StatusPublished Status = "published"
	// Never stored: what the *other* partner sees while the setter writes.
	// It is a view of draft, so the waiting partner learns that a week exists
	// without seeing half-written prayers.
	StatusWaiting Status = "waiting"
)

var (
	ErrNotFound    = apperr.NotFound("prayer_week_not_found", "That prayer week isn’t here.")
	ErrNotSetter   = apperr.Forbidden("not_this_weeks_setter", "It’s your partner’s week to set the prayers.")
	ErrNotDraft    = apperr.Conflict("week_already_published", "This week has been shared already.")
	ErrLockedByUse = apperr.Conflict("week_in_use", "Your partner has started praying these, so they can’t change now.")
	// A week needs two people to have a setter at all, so a couple still
	// waiting for its second member has no week — which is a state of the
	// couple, not a missing thing.
	ErrWaitingForPartner = apperr.Conflict("waiting_for_partner", "Your first prayer week starts when your partner joins.")
)

// Week is one couple's week of prayer.
type Week struct {
	ID           uuid.UUID
	CoupleID     uuid.UUID
	WeekStart    time.Time // the couple's local Sunday, date only
	SetterUserID uuid.UUID
	Status       Status
	PublishedAt  *time.Time
	Points       []Point
}

// Point is one thing to pray about. Body, scripture and verse are optional:
// a title alone ("Ada's interview") is a complete prayer point.
type Point struct {
	ID        uuid.UUID
	Position  int
	Title     string
	Body      string
	Scripture string
	Verse     string
}

// Member is the part of couple membership this module needs: who, and when
// they joined, which is what fixes the order of the rotation.
type Member struct {
	UserID   uuid.UUID
	JoinedAt time.Time
}

// StartOfWeek is the Sunday 00:00 that `at` falls in, in the couple's own
// timezone, as a date-only value.
//
// The couple's timezone, not the server's and not each partner's: partners in
// different places must agree on which week it is, or they'd be praying
// different weeks while sitting next to each other.
func StartOfWeek(at time.Time, loc *time.Location) time.Time {
	local := at.In(loc)
	daysSinceSunday := int(local.Weekday()) // time.Sunday is 0
	y, m, d := local.AddDate(0, 0, -daysSinceSunday).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// WeekIndex counts whole weeks from a couple's first prayer week to this one.
// Both arguments come from StartOfWeek, so this is plain calendar arithmetic
// and daylight saving can't make a week 23 or 25 hours long.
func WeekIndex(firstWeekStart, weekStart time.Time) int {
	return int(weekStart.Sub(firstWeekStart).Hours() / (24 * 7))
}

// SetterFor is whose turn it is: members in join order, alternating every
// week. The result is stored on the week when it is created and never
// recomputed, so the turn can't move retroactively (DEC-18).
//
// It takes the members rather than reading them, so the rotation can be tested
// against any pair without a database.
func SetterFor(members []Member, weekIndex int) (uuid.UUID, error) {
	if len(members) != 2 {
		return uuid.UUID{}, fmt.Errorf("a prayer week needs exactly two members, got %d", len(members))
	}
	ordered := [2]Member{members[0], members[1]}
	if ordered[1].JoinedAt.Before(ordered[0].JoinedAt) {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	// Go's % keeps the sign of the dividend, and a week before the couple's
	// first would be negative, so normalise into {0,1} either way.
	return ordered[((weekIndex%2)+2)%2].UserID, nil
}

// ValidatePoints checks a whole submission and reports every problem at once,
// so the setter doesn't fix one thing only to be told about the next.
// Positions are assigned from the order given: the array *is* the order.
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

// StatusFor is what `viewer` should be told the week's status is. A draft is
// the setter's private workspace; their partner is told a week exists and that
// they're waiting for it, and sees no points (see PointsFor).
func StatusFor(w Week, viewer uuid.UUID) Status {
	if w.Status == StatusDraft && viewer != w.SetterUserID {
		return StatusWaiting
	}
	return w.Status
}

// PointsFor is the week's points as `viewer` may see them: none at all while
// the setter is still writing.
func PointsFor(w Week, viewer uuid.UUID) []Point {
	if w.Status == StatusDraft && viewer != w.SetterUserID {
		return nil
	}
	return w.Points
}

// CanEditPoints reports whether the setter may still change this week.
//
// Two rules in one place: only the setter writes, and once the other partner
// has prayed any of it, it is fixed — editing under someone mid-prayer would
// change what they had already prayed for.
func CanEditPoints(w Week, editor uuid.UUID, partnerHasCompleted bool) error {
	if editor != w.SetterUserID {
		return ErrNotSetter
	}
	if partnerHasCompleted {
		return ErrLockedByUse
	}
	return nil
}

// CanPublish reports whether `publisher` may share this week now. Publishing
// again is not an error: the client may retry, and the second attempt should
// find the world as it wanted it (the service treats it as a no-op).
func CanPublish(w Week, publisher uuid.UUID) error {
	if publisher != w.SetterUserID {
		return ErrNotSetter
	}
	return nil
}
