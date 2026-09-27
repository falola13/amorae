package prayers

import (
	"time"

	"github.com/google/uuid"
)

// Mirrors apps/web/src/lib/api/types.ts. Body is renamed Text to match the
// web's field name; week_end is computed (week_start + 6 days), not stored.
type pointDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Scripture string `json:"scripture,omitempty"`
	Verse     string `json:"verse,omitempty"`
	Position  int    `json:"position"`
	// Which weekdays (0=Sunday..6=Saturday) this point is for; empty means
	// every day, so a point that's for every day never has to say so.
	Weekdays []int `json:"weekdays"`
	// Absent until marked answered; the client keys the answered treatment off this.
	AnsweredAt string `json:"answered_at,omitempty"`
	// answered_at is the instant (for ordering); this is the day in the couple's timezone.
	AnsweredOn string `json:"answered_on,omitempty"`
	AnsweredBy string `json:"answered_by,omitempty"`
	AnswerNote string `json:"answer_note,omitempty"`
}

// dayDTO is one day of the week: which points are scheduled for it, and who
// of the two partners has prayed each of those that day.
type dayDTO struct {
	Date    string   `json:"date"`
	Points  []string `json:"points"`
	Mine    []string `json:"mine"`
	Partner []string `json:"partner"`
}

// answeredDTO carries enough of its week to be placed in time on a screen
// that mixes every week together.
type answeredDTO struct {
	pointDTO
	WeekID    string `json:"week_id"`
	WeekStart string `json:"week_start"`
}

type weekDTO struct {
	ID               string     `json:"id"`
	WeekStart        string     `json:"week_start"`
	WeekEnd          string     `json:"week_end"`
	SetterID         string     `json:"setter_id"`
	Status           Status     `json:"status"`
	Points           []pointDTO `json:"points"`
	MyCompleted      []string   `json:"my_completed"`
	PartnerCompleted []string   `json:"partner_completed"`
	Reflection       string     `json:"reflection,omitempty"`
	// Present only when this week is the one running now — a history week
	// has no "today" of its own.
	Today string `json:"today,omitempty"`
	// Sunday through Saturday, always — a week without a today still has
	// its seven days.
	Days []dayDTO `json:"days"`
}

// ToDTO renders one week as `viewer` is allowed to see it. `today` is the
// couple-local date the caller computed this request at; a zero value, or
// one outside this week's own seven days, means this isn't being viewed as
// the current week, so my_completed/partner_completed fall back to "any day
// this week" (the history view) and Today is left empty.
func ToDTO(rec Record, viewer, partner uuid.UUID, today time.Time) weekDTO {
	status := StatusFor(rec.Week, viewer)
	visible := PointsFor(rec.Week, viewer)

	out := weekDTO{
		ID:               rec.ID.String(),
		WeekStart:        rec.WeekStart.Format(time.DateOnly),
		WeekEnd:          rec.WeekStart.AddDate(0, 0, 6).Format(time.DateOnly),
		SetterID:         rec.SetterUserID.String(),
		Status:           status,
		Points:           make([]pointDTO, 0, len(visible)),
		MyCompleted:      []string{},
		PartnerCompleted: []string{},
		Reflection:       rec.Reflections[viewer],
		Days:             make([]dayDTO, 0, 7),
	}

	for _, p := range visible {
		out.Points = append(out.Points, toPointDTO(p))
	}

	// No visible points means no progress, and no days, to report either.
	if len(visible) == 0 {
		return out
	}

	isCurrent := !today.IsZero() && !today.Before(rec.WeekStart) && today.Before(rec.WeekStart.AddDate(0, 0, 7))
	if isCurrent {
		out.Today = today.Format(time.DateOnly)
		out.MyCompleted = idStrings(rec.ByDay[today][viewer])
		out.PartnerCompleted = idStrings(rec.ByDay[today][partner])
	} else {
		out.MyCompleted = idStrings(rec.Completed[viewer])
		out.PartnerCompleted = idStrings(rec.Completed[partner])
	}

	for i := 0; i < 7; i++ {
		day := rec.WeekStart.AddDate(0, 0, i)
		d := dayDTO{Date: day.Format(time.DateOnly), Points: []string{}, Mine: []string{}, Partner: []string{}}
		for _, p := range visible {
			if ScheduledOn(p.Weekdays, day.Weekday()) {
				d.Points = append(d.Points, p.ID.String())
			}
		}
		d.Mine = idStrings(rec.ByDay[day][viewer])
		d.Partner = idStrings(rec.ByDay[day][partner])
		out.Days = append(out.Days, d)
	}
	return out
}

// ToDTOs renders a list, always as an array rather than null so a client can
// map over it without a nil check. Every record here is history, never the
// current week, so `today` is always the zero value.
func ToDTOs(records []Record, viewer, partner uuid.UUID) []weekDTO {
	out := make([]weekDTO, 0, len(records))
	for _, rec := range records {
		out = append(out, ToDTO(rec, viewer, partner, time.Time{}))
	}
	return out
}

func toPointDTO(p Point) pointDTO {
	out := pointDTO{
		ID:         p.ID.String(),
		Title:      p.Title,
		Text:       p.Body,
		Scripture:  p.Scripture,
		Verse:      p.Verse,
		Position:   p.Position,
		Weekdays:   weekdaysList(p.Weekdays),
		AnswerNote: p.AnswerNote,
	}
	if p.AnsweredAt != nil {
		out.AnsweredAt = p.AnsweredAt.UTC().Format(time.RFC3339)
	}
	if p.AnsweredOn != nil {
		out.AnsweredOn = p.AnsweredOn.Format(time.DateOnly)
	}
	if p.AnsweredBy != (uuid.UUID{}) {
		out.AnsweredBy = p.AnsweredBy.String()
	}
	return out
}

// weekdaysList renders a bitmask as the days it names, empty for
// AllWeekdays — the DTO would rather say nothing than list all seven.
func weekdaysList(mask int) []int {
	out := []int{}
	if mask == AllWeekdays {
		return out
	}
	for d := 0; d <= 6; d++ {
		if mask&(1<<uint(d)) != 0 {
			out = append(out, d)
		}
	}
	return out
}

// ToAnsweredDTOs renders the answered list, always an array rather than null.
func ToAnsweredDTOs(items []Answered) []answeredDTO {
	out := make([]answeredDTO, 0, len(items))
	for _, a := range items {
		out = append(out, answeredDTO{
			pointDTO:  toPointDTO(a.Point),
			WeekID:    a.WeekID.String(),
			WeekStart: a.WeekStart.Format(time.DateOnly),
		})
	}
	return out
}

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
