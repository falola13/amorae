package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// KindNudge is one partner saying they are thinking about the other, with no
// message to write and nothing to reply to.
const KindNudge = "nudge"

// nudgesPerDay is how many one person may send. Enough to be spontaneous,
// few enough that it cannot become a stream.
const nudgesPerDay = 3

var (
	ErrNoPartner = apperr.Conflict("waiting_for_partner",
		"There’s nobody to nudge yet.")
	ErrNudgesSpent = apperr.Conflict("nudges_spent",
		"You’ve sent all of today’s. There’ll be more tomorrow.")
	ErrTheyAreResting = apperr.Conflict("their_quiet_hours",
		"It’s quiet hours where they are. Try again in the morning.")
	ErrTheyHaveHadEnough = apperr.Conflict("their_day_is_full",
		"They’ve had their notifications for today. Try again tomorrow.")
)

// Nudge tells the other partner that this one is thinking about them.
//
// Sent now rather than on the next tick: five minutes late is a different
// thought. And refused rather than queued when it would land badly — being
// told they are asleep is a kinder answer than a notification at 3am, or
// than silence that looks like it worked.
func (s *Service) Nudge(ctx context.Context, senderID uuid.UUID) error {
	partnerID, senderName, err := s.repo.NudgeTarget(ctx, senderID)
	if err != nil {
		return err
	}
	if partnerID == uuid.Nil {
		return ErrNoPartner
	}

	now := s.now()
	budget, err := s.repo.BudgetFor(ctx, partnerID, now)
	if err != nil {
		return err
	}
	zone, zerr := time.LoadLocation(budget.Timezone)
	if zerr != nil {
		zone = time.UTC
	}
	if Quiet(budget.Prefs, zone, now) {
		return ErrTheyAreResting
	}
	if OverCap(budget.Prefs, budget.SentToday) {
		return ErrTheyHaveHadEnough
	}

	// The day's allowance, counted where the person receiving it is, since
	// the rows belong to them.
	midnight := DayStart(zone, now)
	spent, err := s.repo.CountSends(ctx, partnerID, KindNudge, midnight)
	if err != nil {
		return err
	}
	if spent >= nudgesPerDay {
		return ErrNudgesSpent
	}

	n := Notification{
		UserID: partnerID,
		Kind:   KindNudge,
		Key:    fmt.Sprintf("%s:%d", now.In(zone).Format(time.DateOnly), spent+1),
		Message: push.Message{
			Title: "From " + senderName,
			Body:  "Thinking about you.",
			Path:  "/",
			Tag:   KindNudge,
		},
	}
	// Nothing listening is not a failure to report: the thought was sent,
	// and whether a browser was subscribed is not the sender's business.
	if _, err := Deliver(ctx, s.repo, s.sender, s.log, n, now); err != nil {
		return err
	}
	return nil
}
