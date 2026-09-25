package notifications

import (
	"fmt"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// Budget is one person's ceiling, and how close to it they already are.
type Budget struct {
	Prefs     Preferences
	Timezone  string
	SentToday int
}

// Allows reports whether this kind may be sent to this person now, and if
// not, whether it is worth keeping for later.
func (b Budget) Allows(kind string, now time.Time) (send bool, keep bool) {
	zone, err := time.LoadLocation(b.Timezone)
	if err != nil {
		zone = time.UTC
	}
	if Quiet(b.Prefs, zone, now) || OverCap(b.Prefs, b.SentToday) {
		return false, !Perishable(kind)
	}
	return true, true
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
