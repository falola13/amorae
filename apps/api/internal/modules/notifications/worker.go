package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
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
// It names the event: "Breakfast out · Today at 8:30 am". FR-NOTF-005 keeps
// private writing off the lock screen — prayers, journal, appreciation — and
// a plan is not that. It is a calendar entry the two of them made together,
// and a reminder that will not say what it is about is a reminder you have to
// unlock your phone to understand, which is no reminder at all.
//
// Everything else the event holds stays inside: no location, no notes, no
// checklist.
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

	// An untitled event cannot happen through the app, but a notification
	// with an empty headline can, so it has something to fall back on.
	title := strings.TrimSpace(c.Title)
	if title == "" {
		title = "Coming up"
	}

	return Notification{
		UserID: c.UserID,
		Kind:   KindEventReminder,
		// The moment, not just the event: moving something to a new time is
		// asking to be reminded about the new time, and a reminder already
		// sent for the old one should not stop that.
		Key: c.EventID.String() + "@" + at.UTC().Format(time.RFC3339),
		Message: push.Message{
			Title: title,
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

// importantDateLeads are the two moments a kept date is worth saying
// something about, and why there are two.
//
// A week's notice is the one you can act on — book the table, buy the thing,
// take the day off. The morning itself is the one that matters: nobody wants
// to be told about their anniversary only in time to plan it. Each is claimed
// separately, so one being sent never swallows the other.
var importantDateLeads = []struct {
	days int
	name string
}{
	{days: 7, name: "week"},
	{days: 0, name: "day"},
}

// ForImportantDates is what to say, if anything, about the dates this couple
// keeps — a birthday, an anniversary, the day they met.
//
// It returns however many are due, which is nearly always none: a kept date
// is a day of the year, and the question asked of it every tick is whether
// today, or the day a week from today, is that day.
//
// The reminder goes out on the morning, in the couple's zone, and stays due
// for the rest of that day. A prayer reminder missed by an hour can go out
// tomorrow; an anniversary cannot, so any tick that runs at all that day
// catches it.
func ForImportantDates(c ImportantDateCandidate, now time.Time) []Notification {
	if !c.Prefs.ImportantDates || !c.Reminder {
		return nil
	}

	zone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		zone = time.UTC
	}
	local := now.In(zone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	if local.Before(today.Add(reminderMorning * time.Hour)) {
		return nil // still the small hours; nobody needs this yet
	}

	var due []Notification
	for _, lead := range importantDateLeads {
		on := today.AddDate(0, 0, lead.days)
		if !OccursOn(c.Date, on) {
			continue
		}
		due = append(due, Notification{
			UserID: c.UserID,
			Kind:   KindImportantDate,
			// The occurrence, not the date: the same anniversary comes round
			// every year and each year is its own send.
			Key: fmt.Sprintf("%s:%s:%s", c.MilestoneID, on.Format(time.DateOnly), lead.name),
			Message: push.Message{
				Title: c.Title,
				Body:  howFarOff(c.Date, on, lead.days),
				Path:  "/together/milestones",
				Tag:   KindImportantDate,
			},
		})
	}
	return due
}

// howFarOff says when it is and, when the date has a history, how long it has
// been. "Three years today" is the thing worth saying; "2023" is a fact they
// already have.
func howFarOff(date, on time.Time, days int) string {
	years := on.Year() - date.Year()
	if days > 0 {
		if years > 0 {
			return fmt.Sprintf("%s — %s.", plural(years, "year"), on.Format("Monday 2 January"))
		}
		return fmt.Sprintf("In a week — %s.", on.Format("Monday 2 January"))
	}
	if years > 0 {
		return plural(years, "year") + " today."
	}
	return "Today."
}

func plural(n int, unit string) string {
	if n == 1 {
		return "One " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// writtenGrace is how far back the worker looks for something one partner
// wrote for the other.
//
// It is not about being late — the claim row already makes a late send safe
// — but about a first deploy, or a worker that has been down since Tuesday,
// not opening with a week of buzzing about notes somebody has long since
// read.
const writtenGrace = 24 * time.Hour

// ForWritten is the nudge when one of them writes something for the other: a
// note of appreciation, or an entry in the journal.
//
// Only the other partner hears about it. Telling somebody they have written
// something is the emptiest notification there is, and for an appreciation it
// would also undo the point of it.
//
// It waits out the undo window before going anywhere. A note taken back ten
// seconds after it was sent should never have reached a lock screen, and the
// worker is the only thing that can promise that — the send itself cannot
// know what happens next.
func ForWritten(c WrittenCandidate, now time.Time) (Notification, bool) {
	if c.UserID == c.AuthorID {
		return Notification{}, false
	}
	var wanted bool
	var message push.Message
	switch c.Kind {
	case KindAppreciation:
		wanted = c.Prefs.Appreciation
		message = push.Message{
			Title: c.AuthorName + " appreciated you",
			Body:  "A note, just for you.",
			Path:  "/together/appreciation",
			Tag:   KindAppreciation,
		}
	case KindGoal:
		// The goal is named and the amount is not. A goal is a plan the two
		// of them made, like an event (FR-NOTF-005.AC2); what somebody just
		// moved in or out of their savings is not something to put on a
		// lock screen in a coffee shop.
		wanted = c.Prefs.Goals
		message = push.Message{
			Title: c.AuthorName + " put something towards " + c.Subject,
			Body:  "See where the two of you are up to.",
			Path:  "/together/goals",
			Tag:   KindGoal,
		}
	case KindPrayerAnswered:
		// Nothing about which prayer. FR-NOTF-005.AC1 draws its line between
		// a shared plan and private writing, and a prayer point is the
		// second: one person wrote it, and the realistic ones are a parent's
		// illness, a pregnancy, a debt, a marriage under strain. The event
		// exception (AC2) does not reach this, however much more useful a
		// named notification would be — a lock screen in a crowded room is
		// exactly where the cost of being wrong about that lands.
		wanted = c.Prefs.PrayerAnswered
		message = push.Message{
			Title: c.AuthorName + " marked a prayer answered",
			Body:  "Something the two of you prayed for.",
			Path:  "/prayers/answered",
			Tag:   KindPrayerAnswered,
		}
	default:
		wanted = c.Prefs.Journal
		message = push.Message{
			Title: c.AuthorName + " wrote in your journal",
			Body:  "Something they wanted to keep.",
			Path:  "/together/journal",
			Tag:   KindJournal,
		}
	}
	if !wanted {
		return Notification{}, false
	}

	ready := c.WrittenAt.Add(c.Settles)
	if now.Before(ready) || !now.Before(c.WrittenAt.Add(writtenGrace)) {
		return Notification{}, false
	}
	return Notification{
		UserID:  c.UserID,
		Kind:    c.Kind,
		Key:     c.ItemID.String(),
		Message: message,
	}, true
}

// ChallengeCandidate is one person and the challenge their couple is part
// way through.
type ChallengeCandidate struct {
	UserID uuid.UUID
	// The challenge, and what it is called.
	ChallengeID uuid.UUID
	Title       string
	// The couple's zone: which day of the challenge it is is a fact about
	// where they are, not where the server is.
	Timezone string
	// Which day of it today is, counting from one, and how many there are.
	// Worked out in SQL against the couple's own date, because that is where
	// started_on and the zone already sit together.
	Day  int
	Days int
	// Whether this person has already said something about today.
	MarkedToday bool
	Prefs       Preferences
}

// ForChallenge is the daily nudge for a challenge somebody is in the middle
// of and has not marked today.
//
// It goes out in the morning rather than at their prayer reminder time. Those
// are the two recurring nudges in the app, and firing both at seven in the
// evening would make one of them noise.
//
// A challenge is never a streak and this never says how many days were
// missed. Missing yesterday is not a thing to be told about — the whole point
// of the model is that a skipped day is a day, not a failure (DEC-30).
func ForChallenge(c ChallengeCandidate, now time.Time) (Notification, bool) {
	if !c.Prefs.Challenges || c.MarkedToday {
		return Notification{}, false
	}
	// Before it starts, or after the last day: nothing to nudge about.
	if c.Day < 1 || c.Day > c.Days {
		return Notification{}, false
	}

	zone, err := time.LoadLocation(c.Timezone)
	if err != nil {
		zone = time.UTC
	}
	local := now.In(zone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	if local.Before(today.Add(reminderMorning * time.Hour)) {
		return Notification{}, false
	}

	return Notification{
		UserID: c.UserID,
		Kind:   KindChallenge,
		// One a day, keyed on their own date, exactly as the prayer reminder
		// is: a worker that restarts or runs late still sends one.
		Key: c.ChallengeID.String() + ":" + today.Format(time.DateOnly),
		Message: push.Message{
			Title: c.Title,
			Body:  fmt.Sprintf("Day %d of %d is waiting for you.", c.Day, c.Days),
			Path:  "/together/challenges",
			Tag:   KindChallenge,
		},
	}, true
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
	Title   string
	// The couple's zone, not the person's: it is the zone the event's date
	// and time were written in.
	Timezone string
	Date     time.Time
	// "" when the event has no time — a whole day, not a moment.
	StartTime string
	Reminder  string
	Prefs     Preferences
}

// ImportantDateCandidate is one person and one date their couple keeps.
type ImportantDateCandidate struct {
	UserID      uuid.UUID
	MilestoneID uuid.UUID
	Title       string
	// The couple's zone: a date they share is not two dates.
	Timezone string
	Date     time.Time
	// "Remind us every year", as the composer puts it. A date kept without
	// it belongs to their story and is never announced.
	Reminder bool
	Prefs    Preferences
}

// WrittenCandidate is one thing one partner wrote, and the other partner who
// has not been told about it.
type WrittenCandidate struct {
	UserID     uuid.UUID
	AuthorID   uuid.UUID
	AuthorName string
	ItemID     uuid.UUID
	// What the thing is called, when it has a name worth saying: the goal
	// they put something towards. Empty for a journal entry or a note,
	// which have no name and whose words stay inside the app.
	Subject string
	// KindAppreciation, KindJournal or KindGoal.
	Kind      string
	WrittenAt time.Time
	// How long this kind waits before it is safe to announce — the undo
	// window for an appreciation, nothing for a journal entry.
	Settles time.Duration
	Prefs   Preferences
}

// WorkerRepository is what the worker needs of storage.
type WorkerRepository interface {
	CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error)
	DueEventReminders(ctx context.Context, now time.Time) ([]EventCandidate, error)
	ImportantDates(ctx context.Context) ([]ImportantDateCandidate, error)
	RecentlyWritten(ctx context.Context, since time.Time) ([]WrittenCandidate, error)
	LiveChallenges(ctx context.Context) ([]ChallengeCandidate, error)
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

	dates, err := w.repo.ImportantDates(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range dates {
		due = append(due, ForImportantDates(c, now)...)
	}

	written, err := w.repo.RecentlyWritten(ctx, now.Add(-writtenGrace))
	if err != nil {
		return 0, err
	}
	for _, c := range written {
		if n, ok := ForWritten(c, now); ok {
			due = append(due, n)
		}
	}

	challenges, err := w.repo.LiveChallenges(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range challenges {
		if n, ok := ForChallenge(c, now); ok {
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
			w.log.Info("a subscription is gone; removing it", "service", pushService(d.Endpoint))
			if err := w.repo.Unsubscribe(ctx, d.Endpoint); err != nil {
				w.log.Warn("could not remove a dead subscription", "error", err)
			}
		case err != nil:
			// Which push service refused matters more than the error alone:
			// one device failing while another succeeds is the shape of a
			// platform problem, and without this the log cannot tell you
			// which platform.
			w.log.Warn("a device did not take the notification",
				"service", pushService(d.Endpoint), "kind", n.Kind, "error", err)
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

// pushService names the service behind an endpoint — apple, fcm, mozilla —
// without putting the endpoint itself in a log. The rest of the URL is the
// address of one person's browser and belongs in the database, not in
// something we read over somebody's shoulder.
func pushService(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "unknown"
	}
	switch host := u.Host; {
	case strings.Contains(host, "apple"):
		return "apple"
	case strings.Contains(host, "googleapis"), strings.Contains(host, "google"):
		return "fcm"
	case strings.Contains(host, "mozilla"):
		return "mozilla"
	default:
		return host
	}
}

func (w *Worker) release(ctx context.Context, n Notification, cause error) error {
	if err := w.repo.ReleaseSend(ctx, n.UserID, n.Kind, n.Key); err != nil {
		w.log.Warn("could not release a notification claim", "error", err)
	}
	return cause
}
