package prayers

import (
	"time"

	"github.com/google/uuid"
)

// The shape in apps/web/src/lib/api/types.ts. Two things differ from the
// domain on purpose:
//
//   - the domain calls a point's text Body, matching its column; the web has
//     always called it text. A DTO is exactly where that is reconciled, and
//     renaming either side to match the other would be churn for nothing.
//   - week_end is not stored. It is week_start plus six days, every time,
//     so storing it would only create something that can disagree.
type pointDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Scripture string `json:"scripture,omitempty"`
	Verse     string `json:"verse,omitempty"`
	Position  int    `json:"position"`
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

// ToDTO renders one week as `viewer` is allowed to see it.
//
// Everything a viewer must not see is filtered here rather than at the edges:
// a week still being written shows its partner no points, no progress and no
// hint of either. StatusFor and PointsFor make those decisions; this function
// only has to ask them and not leak around the answer.
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
		out.Points = append(out.Points, pointDTO{
			ID:        p.ID.String(),
			Title:     p.Title,
			Text:      p.Body,
			Scripture: p.Scripture,
			Verse:     p.Verse,
			Position:  p.Position,
		})
	}

	// With no points visible there is no progress to report either — saying
	// how many things your partner has prayed for would describe a week you
	// are not allowed to read.
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

func idStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
