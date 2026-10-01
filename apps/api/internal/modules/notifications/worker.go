package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
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

// ForNewWeek notifies only the setter, and only when nothing is set yet —
// telling the other partner would reveal the week isn't done.
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

// ForPublishedWeek notifies whoever didn't publish the week, once, when
// it's ready to read. Either partner may publish now (DEC-33); PublishedBy
// says who did, falling back to the setter for a week from before that, or
// one whose publisher somehow didn't get recorded.
func ForPublishedWeek(c Candidate) (Notification, bool) {
	publisher := c.PublishedBy
	if publisher == uuid.Nil {
		publisher = c.SetterUserID
	}
	if c.UserID == publisher || c.WeekStatus != "published" || c.Points == 0 || !c.Prefs.NewWeek {
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

// ForReminder is the daily nudge for today's unfinished points in a
// published week. Resets every day along with what's scheduled and what's
// been prayed (DEC-33) — the key is the local date, so a day that starts
// with nothing left simply produces no reminder rather than a stale one.
// The body never names what's left — it lands on a lock screen (FR-NOTF-005).
func ForReminder(c Candidate, now time.Time) (Notification, bool, error) {
	if c.WeekStatus != "published" || c.TodayScheduled == 0 || c.TodayCompleted >= c.TodayScheduled || !c.Prefs.PrayerReminder {
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

	left := c.TodayScheduled - c.TodayCompleted
	body := fmt.Sprintf("You have %d left today.", left)
	if left == 1 {
		body = "You have one left today."
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

// eventReminderGrace bounds how late a reminder may still fire, e.g. after
// a worker restart; past it, the event has effectively already happened.
const eventReminderGrace = time.Hour

// ForEventReminder nudges before a planned event and names it — unlike
// prayers, journal, or appreciation, a shared plan isn't private
// (FR-NOTF-005). No location, notes, or checklist leaves the app.
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

	// Defensive fallback; the app itself never allows an untitled event.
	title := strings.TrimSpace(c.Title)
	if title == "" {
		title = "Coming up"
	}

	return Notification{
		UserID: c.UserID,
		Kind:   KindEventReminder,
		// Keyed by moment, not just event: rescheduling should not suppress
		// a new reminder because the old time was already sent.
		Key: c.EventID.String() + "@" + at.UTC().Format(time.RFC3339),
		Message: push.Message{
			Title: title,
			Body:  body,
			Path:  "/together/events/" + c.EventID.String(),
			Tag:   KindEventReminder,
		},
	}, true
}

// whenItIs phrases "Today"/"Tomorrow"/weekday relative to when the reminder
// fires, not to now.
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

// importantDateLeads: a week out (actionable notice) and the day itself.
// Claimed separately so one being sent never swallows the other.
var importantDateLeads = []struct {
	days int
	name string
}{
	{days: 7, name: "week"},
	{days: 0, name: "day"},
}

// ForImportantDates returns whatever dates (birthday, anniversary, etc.) are
// due today or a week from today, in the couple's zone. Stays due for the
// rest of the day, unlike a prayer reminder, so a late tick still catches it.
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
			// Keyed per occurrence: each year's anniversary is its own send.
			Key: fmt.Sprintf("%s:%s:%s", c.MilestoneID, on.Format(time.DateOnly), lead.name),
			Message: push.Message{
				Title: c.Title,
				Body:  howFarOff(c.Date, on, lead.days, c.YearKnown),
				Path:  "/together/milestones",
				Tag:   KindImportantDate,
			},
		})
	}
	return due
}

// howFarOff names the date, plus a "years today" count when there's
// history — never that count for a birthday with no known year, where
// date's year is only milestones' 2000 placeholder and would otherwise
// claim an age nobody actually gave.
func howFarOff(date, on time.Time, days int, yearKnown bool) string {
	years := 0
	if yearKnown {
		years = on.Year() - date.Year()
	}
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

// writtenGrace bounds the backlog window (e.g. a worker down since
// Tuesday), not send timing — the claim row already makes a late send safe.
const writtenGrace = 24 * time.Hour

// ForWritten notifies the other partner once the undo window has passed —
// a note taken back before then should never reach a lock screen.
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
		// Goal named, amount withheld — a shared plan (FR-NOTF-005.AC2), but
		// the dollar figure isn't lock-screen material.
		wanted = c.Prefs.Goals
		message = push.Message{
			Title: c.AuthorName + " put something towards " + c.Subject,
			Body:  "See where the two of you are up to.",
			Path:  "/together/goals",
			Tag:   KindGoal,
		}
	case KindEventAdded:
		// A "together" event a partner just made for the two of them; a
		// "mine" event never reaches this candidate at all (RecentlyWritten
		// only ever selects together ones), and neither does the creator.
		wanted = c.Prefs.PartnerEvents
		message = push.Message{
			Title: c.AuthorName + " added something for you both",
			Body:  c.Subject,
			Path:  "/together/events/" + c.ItemID.String(),
			Tag:   KindEventAdded,
		}
	case KindPrayerAnswered:
		// No prayer content: private writing (FR-NOTF-005.AC1), unlike the
		// event exception (AC2) — wrong here costs more than it saves.
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
	UserID      uuid.UUID
	ChallengeID uuid.UUID
	Title       string
	// The couple's zone; which day it is depends on where they are, not
	// the server.
	Timezone string
	// Day (1-based) and total days; computed in SQL against started_on + zone.
	Day         int
	Days        int
	MarkedToday bool
	Prefs       Preferences
}

// ForChallenge nudges once per morning, not at the prayer reminder time (to
// avoid clustering), and never says how many days were missed (DEC-30).
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
		// Keyed per person per day, not per challenge: however many a couple
		// has running, ForChallenges folds them into one push, so retries
		// send at most once.
		Key: today.Format(time.DateOnly),
		Message: push.Message{
			Title: c.Title,
			Body:  fmt.Sprintf("Day %d of %d is waiting for you.", c.Day, c.Days),
			Path:  "/together/challenges",
			Tag:   KindChallenge,
		},
	}, true
}

// ForChallenges is ForChallenge across everything a couple has running: one
// push per person per day, however many challenges are waiting. One keeps
// its own message; two or more are a single "n challenges today" naming them,
// in the order given (the repository gives them oldest-started first). Each
// still stops by itself after its last day, since ForChallenge decides that.
func ForChallenges(cs []ChallengeCandidate, now time.Time) []Notification {
	var order []uuid.UUID
	waiting := map[uuid.UUID][]Notification{}
	titles := map[uuid.UUID][]string{}
	for _, c := range cs {
		n, ok := ForChallenge(c, now)
		if !ok {
			continue
		}
		if _, seen := waiting[c.UserID]; !seen {
			order = append(order, c.UserID)
		}
		waiting[c.UserID] = append(waiting[c.UserID], n)
		titles[c.UserID] = append(titles[c.UserID], c.Title)
	}

	out := make([]Notification, 0, len(order))
	for _, user := range order {
		n := waiting[user][0]
		if len(waiting[user]) > 1 {
			n.Message.Title = fmt.Sprintf("%d challenges today", len(waiting[user]))
			n.Message.Body = strings.Join(titles[user], " · ")
		}
		out = append(out, n)
	}
	return out
}

// statusDraft mirrors the prayers module's value without importing it — one
// string is cheaper to own than a cross-module dependency.
type weekStatus string

const statusDraft weekStatus = "draft"

// EventCandidate is separate from Candidate: it asks "is anything planned
// about to happen", not "where is this week up to".
type EventCandidate struct {
	UserID  uuid.UUID
	EventID uuid.UUID
	Title   string
	// The couple's zone (the event's own), not the person's.
	Timezone string
	Date     time.Time
	// "" when the event has no time — a whole day, not a moment.
	StartTime string
	EndTime   string
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
	// Whether to announce it yearly; false means kept but never notified.
	Reminder bool
	// False only for a birthday whose profile has no year — Date's year is
	// then milestones' 2000 placeholder, and howFarOff must not turn it
	// into an "N years" claim nobody's age actually supports.
	YearKnown bool
	Prefs     Preferences
}

// WrittenCandidate is one thing one partner wrote, and the other partner who
// has not been told about it.
type WrittenCandidate struct {
	UserID     uuid.UUID
	AuthorID   uuid.UUID
	AuthorName string
	ItemID     uuid.UUID
	// The goal's name, when there is one; empty for journal/appreciation.
	Subject string
	// KindAppreciation, KindJournal or KindGoal.
	Kind      string
	WrittenAt time.Time
	// Undo window before this is safe to announce; zero for a journal entry.
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
	BothMarkedDays(ctx context.Context, since time.Time) ([]BothMarkedCandidate, error)
	BothPrayedWeeks(ctx context.Context, now, since time.Time) ([]BothPrayedCandidate, error)
	MemoriesOnThisDay(ctx context.Context) ([]MemoryAnniversaryCandidate, error)
	EndedEvents(ctx context.Context, now time.Time) ([]EventCandidate, error)
	GoalsJustMovedOn(ctx context.Context, since time.Time) ([]GoalCrossingCandidate, error)
	BudgetFor(ctx context.Context, userID uuid.UUID, now time.Time) (Budget, error)
	ClaimSend(ctx context.Context, userID uuid.UUID, kind, key string, at time.Time) (bool, error)
	ReleaseSend(ctx context.Context, userID uuid.UUID, kind, key string) error
	SubscriptionsFor(ctx context.Context, userID uuid.UUID) ([]Subscription, error)
	Unsubscribe(ctx context.Context, endpoint string) error
	MarkSent(ctx context.Context, endpoint string, at time.Time) error
	RecordInbox(ctx context.Context, userID uuid.UUID, kind, title, body, path string, at time.Time) error
}

// pokeTimeout bounds a pass Poke drives, off its own background context —
// generous like the cron endpoint's tickTimeout, since nothing is waiting on
// a response either way.
const pokeTimeout = 60 * time.Second

// Poker lets a write path outside this module ask for an out-of-band pass,
// without either of them needing to know how the worker is put together.
type Poker interface{ Poke() }

// Worker decides who to tell what, and tells them.
type Worker struct {
	repo   WorkerRepository
	sender push.Sender
	now    func() time.Time
	log    *slog.Logger

	// running makes one pass exclusive — the cron endpoint and Poke both go
	// through TryTick, so a pass either drives can never overlap the
	// other's.
	running sync.Mutex

	pokeMu      sync.Mutex
	pokeRunning bool
	// pokePending is one poke remembered for later, arrived while pokeLoop
	// was already going — coalesced, so however many pile up during a pass,
	// only one more pass follows it.
	pokePending bool
	// afterPoke, when set, runs once after every pass Poke drives. Nobody in
	// production sets it; it is the seam tests use to know a pass has
	// settled without polling for it.
	afterPoke func()

	// wake is a pass booked for the moment the earliest undo window closes
	// (see wakeWhenSettled); wakeFor is when. Guarded by pokeMu.
	wake    *time.Timer
	wakeFor time.Time
}

func NewWorker(repo WorkerRepository, sender push.Sender, now func() time.Time, log *slog.Logger) *Worker {
	return &Worker{repo: repo, sender: sender, now: now, log: log}
}

// TryTick runs one pass if nothing else is already running one, reporting
// whether this call was the one that ran it. Shared by the tick endpoint and
// Poke, through the same lock, so a cron tick and a poked pass can never run
// at once.
func (w *Worker) TryTick(ctx context.Context) (sent int, ran bool, err error) {
	if !w.running.TryLock() {
		return 0, false, nil
	}
	defer w.running.Unlock()
	sent, err = w.Tick(ctx)
	return sent, true, err
}

// Poke asks for one more pass soon, off whatever request triggered it —
// something a partner just did, worth telling them about sooner than the
// next cron tick. Fire-and-forget: it returns at once, and never surfaces an
// error to the caller, who has already finished their own write.
//
// Coalesced: a poke that lands while a pass is already going (whether that
// pass came from another poke or from the cron endpoint, via TryTick) is
// remembered as a single pending poke and runs once more when the current
// pass finishes. However many pokes arrive mid-pass, at most one more pass
// follows — never two passes running together, and never one pass per poke.
func (w *Worker) Poke() {
	w.pokeMu.Lock()
	if w.pokeRunning {
		w.pokePending = true
		w.pokeMu.Unlock()
		return
	}
	w.pokeRunning = true
	w.pokeMu.Unlock()

	go w.pokeLoop()
}

func (w *Worker) pokeLoop() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), pokeTimeout)
		_, _, err := w.TryTick(ctx)
		cancel()
		if err != nil {
			w.log.Error("a poked notification pass failed", "error", err)
		}

		w.pokeMu.Lock()
		again := w.pokePending
		w.pokePending = false
		if !again {
			w.pokeRunning = false
		}
		w.pokeMu.Unlock()

		if hook := w.afterPoke; hook != nil {
			hook()
		}
		if !again {
			return
		}
	}
}

// maxSettleWake bounds wakeWhenSettled to the short undo windows it exists
// for; anything further off is the cron's to find.
const maxSettleWake = 5 * time.Minute

// wakeWhenSettled books one more pass for the moment the earliest undo window
// among these closes. A poke right after an appreciation or an answered
// prayer finds it still inside its window and sends nothing, so without this
// "immediate" meant "at the next cron tick", up to five minutes later —
// for exactly the kinds that have a window.
func (w *Worker) wakeWhenSettled(written []WrittenCandidate, now time.Time) {
	var next time.Time
	for _, c := range written {
		if c.UserID == c.AuthorID || c.Settles <= 0 {
			continue
		}
		ready := c.WrittenAt.Add(c.Settles)
		if ready.After(now) && (next.IsZero() || ready.Before(next)) {
			next = ready
		}
	}
	if next.IsZero() || next.Sub(now) > maxSettleWake {
		return
	}
	w.pokeMu.Lock()
	defer w.pokeMu.Unlock()
	if w.wake != nil && !w.wakeFor.After(next) && w.wakeFor.After(now) {
		return // one is already booked for then or sooner
	}
	if w.wake != nil {
		w.wake.Stop()
	}
	w.wakeFor = next
	// A second past the window, so the pass lands on the far side of it.
	w.wake = time.AfterFunc(next.Sub(now)+time.Second, w.Poke)
}

// Tick does one pass: find who is due, claim each send, and deliver.
// Returns count sent; a failed individual send is logged and skipped.
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
	w.wakeWhenSettled(written, now)

	challenges, err := w.repo.LiveChallenges(ctx)
	if err != nil {
		return 0, err
	}
	due = append(due, ForChallenges(challenges, now)...)

	bothMarked, err := w.repo.BothMarkedDays(ctx, now.Add(-mutualGrace))
	if err != nil {
		return 0, err
	}
	for _, c := range bothMarked {
		if n, ok := ForBothMarked(c); ok {
			due = append(due, n)
		}
	}

	bothPrayed, err := w.repo.BothPrayedWeeks(ctx, now, now.Add(-mutualGrace))
	if err != nil {
		return 0, err
	}
	for _, c := range bothPrayed {
		if n, ok := ForBothPrayed(c); ok {
			due = append(due, n)
		}
	}

	moved, err := w.repo.GoalsJustMovedOn(ctx, now.Add(-mutualGrace))
	if err != nil {
		return 0, err
	}
	for _, c := range moved {
		if n, ok := ForGoalCrossing(c); ok {
			due = append(due, n)
		}
	}

	ended, err := w.repo.EndedEvents(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, c := range ended {
		if n, ok := ForEventOver(c, now); ok {
			due = append(due, n)
		}
	}

	anniversaries, err := w.repo.MemoriesOnThisDay(ctx)
	if err != nil {
		return 0, err
	}
	// Grouped per person: the notification is about the day, not about each
	// moment in it.
	byPerson := map[uuid.UUID][]MemoryAnniversaryCandidate{}
	for _, c := range anniversaries {
		byPerson[c.UserID] = append(byPerson[c.UserID], c)
	}
	for _, theirs := range byPerson {
		if n, ok := ForMemoriesOnThisDay(theirs, now); ok {
			due = append(due, n)
		}
	}

	// One budget per person per tick, so a tick cannot spend more than a day
	// allows by asking the database the same question repeatedly.
	budgets := map[uuid.UUID]Budget{}

	sent := 0
	for _, n := range due {
		if ctx.Err() != nil {
			return sent, ctx.Err()
		}

		budget, known := budgets[n.UserID]
		if !known {
			budget, err = w.repo.BudgetFor(ctx, n.UserID, now)
			if err != nil {
				w.log.Error("reading a notification budget", "user", n.UserID, "error", err)
				continue
			}
			budgets[n.UserID] = budget
		}
		if send, keep, why := budget.Allows(n.Kind, now); !send {
			// Nothing is recorded either way. A kept one is still due on the
			// next tick, once the window opens or the day turns over; a
			// perishable one stops being due by itself.
			//
			// Which rule, and the numbers behind it. "Held" on its own sent
			// somebody reading four files to discover it was the cap, on a
			// day when the cap had just arrived and nobody had chosen it.
			w.log.Info("holding a notification",
				"kind", n.Kind, "user", n.UserID, "keep", keep, "because", why,
				"sent_today", budget.SentToday, "cap", budget.Prefs.DailyCap)
			continue
		}

		ok, err := w.deliver(ctx, n, now)
		if err != nil {
			w.log.Error("sending a notification", "kind", n.Kind, "user", n.UserID, "error", err)
			continue
		}
		if ok {
			sent++
			budget.SentToday++
			budgets[n.UserID] = budget
		}
	}
	return sent, nil
}

func (w *Worker) deliver(ctx context.Context, n Notification, now time.Time) (bool, error) {
	return Deliver(ctx, w.repo, w.sender, w.log, n, now)
}

// pushService names the service behind an endpoint without logging the
// endpoint itself, which identifies one person's browser.
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
