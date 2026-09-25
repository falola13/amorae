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
	// Absent until marked answered; the client keys the answered treatment off this.
	AnsweredAt string `json:"answered_at,omitempty"`
	// answered_at is the instant (for ordering); this is the day in the couple's timezone.
	AnsweredOn string `json:"answered_on,omitempty"`
	AnsweredBy string `json:"answered_by,omitempty"`
	AnswerNote string `json:"answer_note,omitempty"`
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
}

// ToDTO renders one week as `viewer` is allowed to see it. Visibility is
// filtered here (via StatusFor/PointsFor) rather than at the edges.
func ToDTO(rec Record, viewer, partner uuid.UUID) weekDTO {
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
	}

	for _, p := range visible {
		out.Points = append(out.Points, toPointDTO(p))
	}

	// No visible points means no progress to report either.
	if len(visible) == 0 {
		return out
	}
	out.MyCompleted = idStrings(rec.Completed[viewer])
	out.PartnerCompleted = idStrings(rec.Completed[partner])
	return out
}

// ToDTOs renders a list, always as an array rather than null so a client can
// map over it without a nil check.
func ToDTOs(records []Record, viewer, partner uuid.UUID) []weekDTO {
	out := make([]weekDTO, 0, len(records))
	for _, rec := range records {
		out = append(out, ToDTO(rec, viewer, partner))
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
