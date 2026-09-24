package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// Notification is one thing to tell one person, decided before anything is
// sent so the deciding can be tested without a network.
type Notification struct {
	UserID  uuid.UUID
	Kind    string
	Key     string
	Message push.Message
}

// ForNewWeek is what to say, if anything, about a week that has just begun.
//
// Only the setter hears about it, and only when there is something for them
// to do. Telling the other partner "a week has started" would be telling them
// their partner has not written it yet, which is nobody's business and not a
// notification anyone wants.
func ForNewWeek(c Candidate) (Notification, bool) {
	if c.UserID != c.SetterUserID || c.WeekStatus != string(statusDraft) || c.Points > 0 || !c.Prefs.NewWeek {
		return Notification{}, false
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindNewWeek,
		Key:    c.WeekID.String(),
		Message: push.Message{
			Title: "It’s your week",
			Body:  "Set what the two of you will be praying for.",
			Path:  "/prayers/set",
			Tag:   KindNewWeek,
		},
	}, true
}

// ForPublishedWeek tells the other partner their week is ready.
//
// This is the moment the shared thing becomes shared, and the one
// notification that most earns its place — so it goes to the partner who did
// not write it, once, when there is something to read.
func ForPublishedWeek(c Candidate) (Notification, bool) {
	if c.UserID == c.SetterUserID || c.WeekStatus != "published" || c.Points == 0 || !c.Prefs.NewWeek {
		return Notification{}, false
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindWeekPublished,
		Key:    c.WeekID.String(),
		Message: push.Message{
			Title: "Your prayer week is ready",
			Body:  "See what the two of you will be praying for.",
			Path:  "/prayers",
			Tag:   KindWeekPublished,
		},
	}, true
}

// ForReminder is the daily nudge, for someone with a published week they have
// not finished.
//
// The body says how many are left and never what they are: this lands on a
// lock screen, which is the one place in Amorae that is not private
// (FR-NOTF-005).
func ForReminder(c Candidate, now time.Time) (Notification, bool, error) {
	if c.WeekStatus != "published" || c.Points == 0 || c.Completed >= c.Points || !c.Prefs.PrayerReminder {
		return Notification{}, false, nil
	}

	zone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		zone = time.UTC
	}
	date, passed, err := ReminderPassed(c.Prefs.ReminderTime, zone, now)
	if err != nil || !passed {
		return Notification{}, false, err
	}

	left := c.Points - c.Completed
	body := fmt.Sprintf("You have %d left this week.", left)
	if left == 1 {
		body = "You have one left this week."
	}
	return Notification{
		UserID: c.UserID,
		Kind:   KindPrayerReminder,
		Key:    date,
		Message: push.Message{
			Title: "A moment to pray",
			Body:  body,
			Path:  "/prayers",
			Tag:   KindPrayerReminder,
		},
	}, true, nil
}

// eventReminderGrace is how late a reminder may arrive and still be one.
//
// It exists because the worker can be restarted, deployed or simply down.
// Within the hour a nudge is still useful — the body says when the thing is,
// not how long until it — and past it, telling somebody about a coffee that
// started ninety minutes ago is noise.
const eventReminderGrace = time.Hour

// ForEventReminder is the nudge before something the two of them planned.
//
// The body says when, never what. An event title is theirs, and this lands on
// a lock screen, which is the one place in Amorae that is not private
// (FR-NOTF-005) — so "Coming up · Today at 8:30 am", and the event itself is
// one tap away.
func ForEventReminder(c EventCandidate, now time.Time) (Notification, bool) {
	if !c.Prefs.EventReminders {
		return Notification{}, false
	}

	zone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		zone = time.UTC
	}
	at, ok := EventReminderAt(c.Date, c.StartTime, c.Reminder, zone)
	if !ok {
		return Notification{}, false
	}
	if now.Before(at) || !now.Before(at.Add(eventReminderGrace)) {
		return Notification{}, false
	}

	day := time.Date(c.Date.Year(), c.Date.Month(), c.Date.Day(), 0, 0, 0, 0, zone)
	body := whenItIs(day, c.StartTime, at, zone)

	return Notification{
		UserID: c.UserID,
		Kind:   KindEventReminder,
		// The moment, not just the event: moving something to a new time is
		// asking to be reminded about the new time, and a reminder already
		// sent for the old one should not stop that.
		Key: c.EventID.String() + "@" + at.UTC().Format(time.RFC3339),
		Message: push.Message{
			Title: "Coming up",
			Body:  body,
			Path:  "/together/events/" + c.EventID.String(),
			Tag:   KindEventReminder,
		},
	}, true
}

// whenItIs says when the event is, from where the reminder is standing.
func whenItIs(day time.Time, startTime string, at time.Time, zone *time.Location) string {
	when := "Today"
	switch days := int(day.Sub(time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, zone)).Hours() / 24); {
	case days == 1:
		when = "Tomorrow"
	case days > 1:
		when = day.Format("Monday")
	}
	if start, timed := startOf(day, startTime, zone); timed {
		return fmt.Sprintf("%s at %s.", when, start.Format("3:04 pm"))
	}
	return when + "."
}

// statusDraft mirrors the prayers module's value without importing it: the
// worker reads the column, and one string is a smaller thing to owe another
// module than a dependency is.
type weekStatus string

const statusDraft weekStatus = "draft"

// EventCandidate is one person and one event of theirs that might be worth a
// nudge. Separate from Candidate because it answers a different question:
// that one is "where is this couple's week up to", this one is "is anything
// they planned about to happen".
type EventCandidate struct {
	UserID  uuid.UUID
	EventID uuid.UUID
	// The couple's zone, not the person's: it is the zone the event's date
	// and time were written in.
	Timezone string
	Date     time.Time
	// "" when the event has no time — a whole day, not a moment.
	StartTime string
	Reminder  string
	Prefs     Preferences
}

// WorkerRepository is what the worker needs of storage.
type WorkerRepository interface {
	CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error)
	DueEventReminders(ctx context.Context, now time.Time) ([]EventCandidate, error)
	ClaimSend(ctx context.Context, userID uuid.UUID, kind, key string, at time.Time) (bool, error)
	ReleaseSend(ctx context.Context, userID uuid.UUID, kind, key string) error
	SubscriptionsFor(ctx context.Context, userID uuid.UUID) ([]Subscription, error)
	Unsubscribe(ctx context.Context, endpoint string) error
	MarkSent(ctx context.Context, endpoint string, at time.Time) error
}

// Worker decides who to tell what, and tells them.
type Worker struct {
	repo   WorkerRepository
	sender push.Sender
	now    func() time.Time
	log    *slog.Logger
}

func NewWorker(repo WorkerRepository, sender push.Sender, now func() time.Time, log *slog.Logger) *Worker {
	return &Worker{repo: repo, sender: sender, now: now, log: log}
}

// Tick does one pass. It is the whole job: find who is due, claim each send
// so nobody else makes it, and deliver.
//
// It returns how many were sent, and an error only for something that stops
// the pass — one person's failed send is logged and skipped, because the
// others are still owed theirs.
func (w *Worker) Tick(ctx context.Context) (int, error) {
	now := w.now()

	candidates, err := w.repo.CurrentWeekCandidates(ctx, now)
	if err != nil {
		return 0, err
	}

	var due []Notification
	for _, c := range candidates {
		if n, ok := ForNewWeek(c); ok {
			due = append(due, n)
		}
		if n, ok := ForPublishedWeek(c); ok {
			due = append(due, n)
		}
		n, ok, err := ForReminder(c, now)
		if err != nil {
			w.log.Warn("skipping a reminder", "user", c.UserID, "error", err)
			continue
		}
		if ok {
			due = append(due, n)
		}
	}

	events, err := w.repo.DueEventReminders(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, c := range events {
		if n, ok := ForEventReminder(c, now); ok {
			due = append(due, n)
		}
	}

	sent := 0
	for _, n := range due {
		if ctx.Err() != nil {
			return sent, ctx.Err()
		}
		ok, err := w.deliver(ctx, n, now)
		if err != nil {
			w.log.Error("sending a notification", "kind", n.Kind, "user", n.UserID, "error", err)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// deliver claims the notification and sends it to every browser this person
// has. A subscription the push service says is gone is deleted rather than
// retried (FR-NOTF-004).
func (w *Worker) deliver(ctx context.Context, n Notification, now time.Time) (bool, error) {
	claimed, err := w.repo.ClaimSend(ctx, n.UserID, n.Kind, n.Key, now)
	if err != nil {
		return false, err
	}
	if !claimed {
		return false, nil // somebody already sent this one
	}

	devices, err := w.repo.SubscriptionsFor(ctx, n.UserID)
	if err != nil {
		return false, w.release(ctx, n, err)
	}
	if len(devices) == 0 {
		// Nowhere to send it. The claim stands: when they do subscribe, this
		// moment has passed, and a week-old "it's your week" helps nobody.
		return false, nil
	}

	delivered := false
	for _, d := range devices {
		err := w.sender.Send(ctx, push.Device{Endpoint: d.Endpoint, P256dh: d.P256dh, Auth: d.Auth}, n.Message)
		switch {
		case errors.Is(err, push.ErrGone):
			if err := w.repo.Unsubscribe(ctx, d.Endpoint); err != nil {
				w.log.Warn("could not remove a dead subscription", "error", err)
			}
		case err != nil:
			w.log.Warn("a device did not take the notification", "error", err)
		default:
			delivered = true
			if err := w.repo.MarkSent(ctx, d.Endpoint, now); err != nil {
				w.log.Warn("could not record a send", "error", err)
			}
		}
	}

	if !delivered {
		// Every device failed, and they may all work in an hour. Let the next
		// tick try again rather than silently swallowing the notification.
		return false, w.release(ctx, n, nil)
	}
	return true, nil
}

func (w *Worker) release(ctx context.Context, n Notification, cause error) error {
	if err := w.repo.ReleaseSend(ctx, n.UserID, n.Kind, n.Key); err != nil {
		w.log.Warn("could not release a notification claim", "error", err)
	}
	return cause
}
