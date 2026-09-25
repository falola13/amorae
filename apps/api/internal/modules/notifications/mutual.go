package notifications

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// Budget is one person's ceiling, and how close to it they already are.
type Budget struct {
	Prefs     Preferences
	Timezone  string
	SentToday int
}

// Allows reports whether this kind may be sent to this person now, whether it
// is worth keeping for later, and — when it is not sent — which rule stopped
// it. The reason comes from the same branch that makes the decision, so a log
// line cannot disagree with what actually happened.
func (b Budget) Allows(kind string, now time.Time) (send bool, keep bool, why string) {
	zone, err := time.LoadLocation(b.Timezone)
	if err != nil {
		zone = time.UTC
	}
	switch {
	case Quiet(b.Prefs, zone, now):
		return false, !Perishable(kind), "quiet_hours"
	case OverCap(b.Prefs, b.SentToday):
		return false, !Perishable(kind), "daily_cap"
	}
	return true, true, ""
}

// ForBothMarked is the second of the two of them finishing a day. Sent to
// both, once each, and only then — one partner marking a day says nothing
// about the other (DEC-30), so this is the only moment that belongs to them
// jointly.
func ForBothMarked(c BothMarkedCandidate) (Notification, bool) {
	if !c.Prefs.Together {
		return Notification{}, false
	}
	body := fmt.Sprintf("You’ve both done day %d of %s.", c.Day, c.Title)
	if c.Day == c.Days {
		body = fmt.Sprintf("You’ve both finished %s.", c.Title)
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindBothMarked,
		Key:    fmt.Sprintf("%s:%d", c.ChallengeID, c.Day),
		Message: push.Message{
			Title: "Both of you",
			Body:  body,
			Path:  "/together/challenges",
			Tag:   KindBothMarked,
		},
	}, true
}

// ForBothPrayed is both of them having prayed everything in the week.
func ForBothPrayed(c BothPrayedCandidate) (Notification, bool) {
	if !c.Prefs.Together {
		return Notification{}, false
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindBothPrayed,
		Key:    c.WeekID.String(),
		Message: push.Message{
			Title: "Both of you",
			Body:  "You’ve both prayed everything this week.",
			Path:  "/prayers",
			Tag:   KindBothPrayed,
		},
	}, true
}

// mutualGrace bounds how far back a finish counts, so shipping this does not
// announce every day either of them ever completed.
const mutualGrace = 24 * time.Hour

// ForMemoriesOnThisDay offers back what this couple kept on this day in an
// earlier year.
//
// One notification for the day rather than one per moment: three
// anniversaries falling together is a lovely thing to open, and three
// separate buzzes is not. Keyed by the date, so it arrives once however
// many there are and however often the worker runs.
//
// `all` is one person's candidates, already narrowed to their couple.
func ForMemoriesOnThisDay(all []MemoryAnniversaryCandidate, now time.Time) (Notification, bool) {
	if len(all) == 0 || !all[0].Prefs.Memories {
		return Notification{}, false
	}

	zone, err := time.LoadLocation(all[0].Timezone)
	if err != nil {
		zone = time.UTC
	}
	local := now.In(zone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	if local.Before(today.Add(reminderMorning * time.Hour)) {
		return Notification{}, false
	}

	var on []MemoryAnniversaryCandidate
	for _, c := range all {
		if OccursOn(c.Date, today) {
			on = append(on, c)
		}
	}
	if len(on) == 0 {
		return Notification{}, false
	}
	// Oldest first: the one furthest back is the one worth naming.
	sort.Slice(on, func(i, j int) bool { return on[i].Date.Before(on[j].Date) })

	oldest := on[0]
	body := oldest.Title
	if len(on) > 1 {
		body = fmt.Sprintf("%s, and %d more.", oldest.Title, len(on)-1)
	}
	return Notification{
		UserID: oldest.UserID,
		Kind:   KindMemoryOnThisDay,
		Key:    today.Format(time.DateOnly),
		Message: push.Message{
			Title: yearsAgo(today.Year() - oldest.Date.Year()),
			Body:  body,
			Path:  "/together/memories",
			Tag:   KindMemoryOnThisDay,
		},
	}, true
}

// yearsAgo reads as a person would say it.
func yearsAgo(years int) string {
	switch {
	case years <= 1:
		return "A year ago today"
	default:
		return fmt.Sprintf("%d years ago today", years)
	}
}

// eventOverGrace bounds how late the question may be asked. A day after is
// still a fair question; a week after is somebody rummaging.
const eventOverGrace = 24 * time.Hour

// ForEventOver asks, once, whether an event that has just finished is worth
// keeping. It leads to the event, where keeping it is already the first
// thing offered once it is over.
//
// Not perishable: asked at eleven at night it waits until morning rather
// than being dropped, because the question keeps.
func ForEventOver(c EventCandidate, now time.Time) (Notification, bool) {
	if !c.Prefs.EventReminders {
		return Notification{}, false
	}
	zone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		zone = time.UTC
	}
	over := EventOverAt(c.Date, c.StartTime, c.EndTime, zone)
	if now.Before(over) || !now.Before(over.Add(eventOverGrace)) {
		return Notification{}, false
	}

	title := strings.TrimSpace(c.Title)
	if title == "" {
		title = "That thing you did"
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindEventOver,
		// Once per event, ever.
		Key: c.EventID.String(),
		Message: push.Message{
			Title: "How was it?",
			Body:  title + " — keep it as a memory.",
			Path:  "/together/events/" + c.EventID.String(),
			Tag:   KindEventOver,
		},
	}, true
}

// goalCrossings are the two points in a goal's life worth interrupting for.
// Not every tenth: a bar that announces itself constantly is a bar nobody
// looks at.
var goalCrossings = []struct {
	percent int
	body    string
}{
	{50, "You’re halfway there."},
	{100, "You’ve reached it."},
}

// ForGoalCrossing announces the furthest point a goal has just passed.
//
// The highest one only: a single contribution that takes a goal from nothing
// to finished is one piece of news, not two. Keyed by the crossing, so
// passing halfway, slipping back and passing it again says nothing the
// second time — which is right, because it is not news the second time.
//
// The amount is left out on purpose (FR-NOTF-005.AC2): the goal is shared,
// the figure is not lock-screen material.
func ForGoalCrossing(c GoalCrossingCandidate) (Notification, bool) {
	if !c.Prefs.GoalMilestones || c.Target <= 0 {
		return Notification{}, false
	}
	reached := -1
	var body string
	for _, crossing := range goalCrossings {
		if c.Total*100 >= c.Target*int64(crossing.percent) {
			reached, body = crossing.percent, crossing.body
		}
	}
	if reached < 0 {
		return Notification{}, false
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindGoalCrossing,
		Key:    fmt.Sprintf("%s:%d", c.GoalID, reached),
		Message: push.Message{
			Title: c.Title,
			Body:  body,
			Path:  "/together/goals",
			Tag:   KindGoalCrossing,
		},
	}, true
}
