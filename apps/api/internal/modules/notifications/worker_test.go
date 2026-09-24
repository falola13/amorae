package notifications

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestForNewWeek(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	week := uuid.New()
	base := Candidate{
		UserID: setter, SetterUserID: setter, WeekID: week,
		WeekStatus: "draft", Points: 0,
		Prefs: Preferences{NewWeek: true},
	}

	t.Run("the setter is told it is their turn", func(t *testing.T) {
		n, ok := ForNewWeek(base)
		if !ok {
			t.Fatal("the setter was told nothing")
		}
		if n.Key != week.String() {
			t.Errorf("key = %q, want the week id", n.Key)
		}
		if n.Message.Path != "/prayers/set" {
			t.Errorf("path = %q, should open the screen where they do the thing", n.Message.Path)
		}
	})

	t.Run("the partner is not", func(t *testing.T) {
		// Telling them "a week has started" is telling them their partner
		// hasn't written it yet, which is nobody's business.
		c := base
		c.UserID = partner
		if _, ok := ForNewWeek(c); ok {
			t.Error("the partner was notified about a week they cannot see")
		}
	})

	t.Run("nobody is nagged about a week they have already written", func(t *testing.T) {
		c := base
		c.Points = 3
		if _, ok := ForNewWeek(c); ok {
			t.Error("the setter was told to set a week they had already set")
		}
	})

	t.Run("nor about one already published", func(t *testing.T) {
		c := base
		c.WeekStatus = "published"
		if _, ok := ForNewWeek(c); ok {
			t.Error("a published week produced a 'set your week' notification")
		}
	})
}

func TestForReminder(t *testing.T) {
	// 19:00 Lagos is 18:00 UTC; this instant is after it.
	now := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	base := Candidate{
		UserID:     uuid.New(),
		Timezone:   "Africa/Lagos",
		Prefs:      Preferences{PrayerReminder: true, ReminderTime: "19:00", NewWeek: true},
		WeekStatus: "published",
		Points:     3,
		Completed:  1,
	}

	t.Run("someone with prayers left is reminded", func(t *testing.T) {
		n, ok, err := ForReminder(base, now)
		if err != nil || !ok {
			t.Fatalf("no reminder: ok=%v err=%v", ok, err)
		}
		if n.Key != "2026-09-23" {
			t.Errorf("key = %q, want their own local date", n.Key)
		}
		if !strings.Contains(n.Message.Body, "2") {
			t.Errorf("body = %q, should say how many are left", n.Message.Body)
		}
	})

	t.Run("the body never says what the prayers are", func(t *testing.T) {
		// It lands on a lock screen (FR-NOTF-005).
		n, _, _ := ForReminder(base, now)
		for _, word := range []string{"interview", "Ada", "mum"} {
			if strings.Contains(n.Message.Body+n.Message.Title, word) {
				t.Errorf("a notification leaked content: %q", n.Message.Body)
			}
		}
	})

	t.Run("one left reads as words, not a digit", func(t *testing.T) {
		c := base
		c.Completed = 2
		n, _, _ := ForReminder(c, now)
		if !strings.Contains(n.Message.Body, "one left") {
			t.Errorf("body = %q", n.Message.Body)
		}
	})

	t.Run("somebody who has finished is left alone", func(t *testing.T) {
		c := base
		c.Completed = c.Points
		if _, ok, _ := ForReminder(c, now); ok {
			t.Error("a finished week still nagged")
		}
	})

	t.Run("an unpublished week has nothing to pray", func(t *testing.T) {
		c := base
		c.WeekStatus = "draft"
		if _, ok, _ := ForReminder(c, now); ok {
			t.Error("a draft week produced a reminder")
		}
	})

	t.Run("before their own reminder time, nothing", func(t *testing.T) {
		early := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC) // 11:00 in Lagos
		if _, ok, _ := ForReminder(base, early); ok {
			t.Error("the reminder fired hours early")
		}
	})
}

// fakeRepo records what the worker did, and can be told to fail.
type fakeRepo struct {
	candidates []Candidate
	events     []EventCandidate
	dates      []ImportantDateCandidate
	written    []WrittenCandidate
	challenges []ChallengeCandidate
	subs       []Subscription
	claimed    map[string]bool
	released   []string
	removed    []string
	sentTo     []string
}

func newFakeRepo() *fakeRepo { return &fakeRepo{claimed: map[string]bool{}} }

func (f *fakeRepo) CurrentWeekCandidates(context.Context, time.Time) ([]Candidate, error) {
	return f.candidates, nil
}
func (f *fakeRepo) DueEventReminders(context.Context, time.Time) ([]EventCandidate, error) {
	return f.events, nil
}
func (f *fakeRepo) ImportantDates(context.Context) ([]ImportantDateCandidate, error) {
	return f.dates, nil
}
func (f *fakeRepo) RecentlyWritten(context.Context, time.Time) ([]WrittenCandidate, error) {
	return f.written, nil
}
func (f *fakeRepo) LiveChallenges(context.Context) ([]ChallengeCandidate, error) {
	return f.challenges, nil
}
func (f *fakeRepo) ClaimSend(_ context.Context, userID uuid.UUID, kind, key string, _ time.Time) (bool, error) {
	k := userID.String() + kind + key
	if f.claimed[k] {
		return false, nil
	}
	f.claimed[k] = true
	return true, nil
}
func (f *fakeRepo) ReleaseSend(_ context.Context, userID uuid.UUID, kind, key string) error {
	k := userID.String() + kind + key
	delete(f.claimed, k)
	f.released = append(f.released, k)
	return nil
}
func (f *fakeRepo) SubscriptionsFor(context.Context, uuid.UUID) ([]Subscription, error) {
	return f.subs, nil
}
func (f *fakeRepo) Unsubscribe(_ context.Context, endpoint string) error {
	f.removed = append(f.removed, endpoint)
	return nil
}
func (f *fakeRepo) MarkSent(_ context.Context, endpoint string, _ time.Time) error {
	f.sentTo = append(f.sentTo, endpoint)
	return nil
}

type fakeSender struct {
	err  error
	sent []push.Message
}

func (s *fakeSender) Send(_ context.Context, _ push.Device, m push.Message) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, m)
	return nil
}

func workerFor(repo WorkerRepository, sender push.Sender) *Worker {
	now := func() time.Time { return time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC) }
	return NewWorker(repo, sender, now, quietLog())
}

func TestWorkerTick(t *testing.T) {
	setter := uuid.New()
	week := uuid.New()
	candidate := Candidate{
		UserID: setter, SetterUserID: setter, WeekID: week, WeekStatus: "draft",
		Prefs: Preferences{NewWeek: true},
	}
	device := Subscription{Endpoint: "https://push.example/abc", P256dh: "k", Auth: "a"}

	t.Run("it sends, once", func(t *testing.T) {
		repo := newFakeRepo()
		repo.candidates = []Candidate{candidate}
		repo.subs = []Subscription{device}
		sender := &fakeSender{}
		w := workerFor(repo, sender)

		sent, err := w.Tick(context.Background())
		if err != nil {
			t.Fatalf("Tick: %v", err)
		}
		if sent != 1 || len(sender.sent) != 1 {
			t.Fatalf("sent = %d, messages = %d, want 1 and 1", sent, len(sender.sent))
		}

		// The whole reason for the claim: a second tick within the same
		// window must not tell them again.
		sent, err = w.Tick(context.Background())
		if err != nil {
			t.Fatalf("second Tick: %v", err)
		}
		if sent != 0 || len(sender.sent) != 1 {
			t.Errorf("the second tick sent it again: sent=%d messages=%d", sent, len(sender.sent))
		}
	})

	t.Run("a dead subscription is forgotten, not retried", func(t *testing.T) {
		repo := newFakeRepo()
		repo.candidates = []Candidate{candidate}
		repo.subs = []Subscription{device}
		w := workerFor(repo, &fakeSender{err: push.ErrGone})

		if _, err := w.Tick(context.Background()); err != nil {
			t.Fatalf("Tick: %v", err)
		}
		if len(repo.removed) != 1 || repo.removed[0] != device.Endpoint {
			t.Errorf("the gone subscription was not removed: %v", repo.removed)
		}
	})

	t.Run("a send that merely failed is tried again next tick", func(t *testing.T) {
		repo := newFakeRepo()
		repo.candidates = []Candidate{candidate}
		repo.subs = []Subscription{device}
		sender := &fakeSender{err: errors.New("push service had a moment")}
		w := workerFor(repo, sender)

		if _, err := w.Tick(context.Background()); err != nil {
			t.Fatalf("Tick: %v", err)
		}
		if len(repo.released) != 1 {
			t.Fatal("the claim was kept, so the notification is lost forever")
		}

		// It works this time.
		sender.err = nil
		sent, err := w.Tick(context.Background())
		if err != nil {
			t.Fatalf("second Tick: %v", err)
		}
		if sent != 1 {
			t.Errorf("sent = %d after recovering, want 1", sent)
		}
	})

	t.Run("nobody subscribed is not an error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.candidates = []Candidate{candidate}
		w := workerFor(repo, &fakeSender{})

		sent, err := w.Tick(context.Background())
		if err != nil {
			t.Fatalf("Tick: %v", err)
		}
		if sent != 0 {
			t.Errorf("sent = %d with no devices", sent)
		}
	})
}

func TestForPublishedWeek(t *testing.T) {
	setter, partner := uuid.New(), uuid.New()
	base := Candidate{
		UserID: partner, SetterUserID: setter, WeekID: uuid.New(),
		WeekStatus: "published", Points: 3,
		Prefs: Preferences{NewWeek: true},
	}

	t.Run("the partner is told it is ready", func(t *testing.T) {
		n, ok := ForPublishedWeek(base)
		if !ok {
			t.Fatal("the partner was not told")
		}
		if n.Message.Path != "/prayers" {
			t.Errorf("path = %q", n.Message.Path)
		}
	})

	t.Run("the setter is not told about their own week", func(t *testing.T) {
		c := base
		c.UserID = setter
		if _, ok := ForPublishedWeek(c); ok {
			t.Error("the setter was told about the week they just published")
		}
	})

	t.Run("an unpublished week says nothing", func(t *testing.T) {
		c := base
		c.WeekStatus = "draft"
		if _, ok := ForPublishedWeek(c); ok {
			t.Error("a draft was announced")
		}
	})

	t.Run("a published week with nothing in it says nothing", func(t *testing.T) {
		c := base
		c.Points = 0
		if _, ok := ForPublishedWeek(c); ok {
			t.Error("an empty week was announced")
		}
	})
}

func TestPreferencesAreRespected(t *testing.T) {
	// A switch that is off has to actually stop the notification, or the
	// settings screen is decoration (FR-NOTF-007.AC1).
	setter, partner := uuid.New(), uuid.New()
	week := uuid.New()

	t.Run("new_week off silences both week notifications", func(t *testing.T) {
		off := Preferences{NewWeek: false, PrayerReminder: true, ReminderTime: "19:00"}
		if _, ok := ForNewWeek(Candidate{
			UserID: setter, SetterUserID: setter, WeekID: week, WeekStatus: "draft", Prefs: off,
		}); ok {
			t.Error("the setter was told despite turning it off")
		}
		if _, ok := ForPublishedWeek(Candidate{
			UserID: partner, SetterUserID: setter, WeekID: week,
			WeekStatus: "published", Points: 2, Prefs: off,
		}); ok {
			t.Error("the partner was told despite turning it off")
		}
	})

	t.Run("prayer_reminder off silences the daily nudge", func(t *testing.T) {
		off := Preferences{NewWeek: true, PrayerReminder: false, ReminderTime: "19:00"}
		_, ok, err := ForReminder(Candidate{
			UserID: partner, SetterUserID: setter, WeekID: week, Timezone: "Africa/Lagos",
			WeekStatus: "published", Points: 3, Completed: 0, Prefs: off,
		}, time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("ForReminder: %v", err)
		}
		if ok {
			t.Error("a reminder fired for somebody who turned reminders off")
		}
	})
}

func TestEventReminderAt(t *testing.T) {
	lagos, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		t.Skip("no timezone database here")
	}
	// Friday.
	date := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		start    string
		reminder string
		want     string // local, "2006-01-02 15:04"
		none     bool
	}{
		{name: "at the time", start: "19:30", reminder: "at the time", want: "2026-09-25 19:30"},
		{name: "minutes", start: "19:30", reminder: "30 minutes before", want: "2026-09-25 19:00"},
		{name: "hours", start: "19:30", reminder: "2 hours before", want: "2026-09-25 17:30"},
		{name: "across midnight", start: "00:30", reminder: "2 hours before", want: "2026-09-24 22:30"},
		{name: "the morning of", start: "19:30", reminder: "the morning of", want: "2026-09-25 08:00"},
		{name: "the day before", start: "19:30", reminder: "1 day before", want: "2026-09-24 08:00"},
		// A phrase somebody typed before the list existed.
		{name: "an hour before", start: "19:30", reminder: "an hour before", want: "2026-09-25 18:30"},
		{name: "case and spacing", start: "19:30", reminder: "  1 Hour Before ", want: "2026-09-25 18:30"},
		// Nothing to be an hour before, so it becomes the morning rather
		// than being dropped.
		{name: "all-day event", start: "", reminder: "1 hour before", want: "2026-09-25 08:00"},
		{name: "all-day, at the time", start: "", reminder: "at the time", want: "2026-09-25 08:00"},
		{name: "all-day, day before", start: "", reminder: "1 day before", want: "2026-09-24 08:00"},
		{name: "no reminder", start: "19:30", reminder: "", none: true},
		{name: "a phrase we cannot read", start: "19:30", reminder: "when you get a chance", none: true},
		{name: "zero is not a lead", start: "19:30", reminder: "0 hours before", none: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			at, ok := EventReminderAt(date, tc.start, tc.reminder, lagos)
			if tc.none {
				if ok {
					t.Fatalf("got %v, want no reminder at all", at)
				}
				return
			}
			if !ok {
				t.Fatal("no reminder, want one")
			}
			if got := at.In(lagos).Format("2006-01-02 15:04"); got != tc.want {
				t.Errorf("at = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestForEventReminder(t *testing.T) {
	if _, err := time.LoadLocation("Africa/Lagos"); err != nil {
		t.Skip("no timezone database here")
	}
	id := uuid.New()
	base := EventCandidate{
		UserID: uuid.New(), EventID: id, Title: "Dinner at Terra",
		Timezone: "Africa/Lagos",
		Date:     time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		// 19:30 Lagos is 18:30 UTC, so "1 hour before" is 17:30 UTC.
		StartTime: "19:30", Reminder: "1 hour before",
		Prefs: Preferences{EventReminders: true},
	}
	due := time.Date(2026, 9, 25, 17, 30, 0, 0, time.UTC)

	t.Run("it goes out once the moment arrives", func(t *testing.T) {
		n, ok := ForEventReminder(base, due)
		if !ok {
			t.Fatal("nothing was sent at the moment it was due")
		}
		if n.Message.Path != "/together/events/"+id.String() {
			t.Errorf("path = %q, should open the event", n.Message.Path)
		}
		// A plan is not private writing: the reminder says what it is for,
		// or there is no point waking somebody for it.
		if n.Message.Title != "Dinner at Terra" {
			t.Errorf("title = %q, should name the event", n.Message.Title)
		}
		if !strings.Contains(n.Message.Body, "7:30 pm") {
			t.Errorf("body = %q, should say when it is", n.Message.Body)
		}
		if !strings.Contains(n.Message.Body, "Today") {
			t.Errorf("body = %q, should place the day", n.Message.Body)
		}
	})

	t.Run("an event with no title still has a headline", func(t *testing.T) {
		c := base
		c.Title = "   "
		n, ok := ForEventReminder(c, due)
		if !ok || n.Message.Title == "" {
			t.Errorf("title = %q, want something to show", n.Message.Title)
		}
	})

	t.Run("nothing else about the event goes out", func(t *testing.T) {
		// Where it is and what is on its checklist stay inside the app.
		n, _ := ForEventReminder(base, due)
		if strings.Contains(n.Message.Body, "Terra") {
			t.Errorf("body = %q, should be the when and nothing more", n.Message.Body)
		}
	})

	t.Run("not a minute before", func(t *testing.T) {
		if _, ok := ForEventReminder(base, due.Add(-time.Minute)); ok {
			t.Error("a reminder went out early")
		}
	})

	t.Run("a late tick still catches it", func(t *testing.T) {
		if _, ok := ForEventReminder(base, due.Add(59*time.Minute)); !ok {
			t.Error("a reminder was lost to a tick that ran late")
		}
	})

	t.Run("but not hours later", func(t *testing.T) {
		if _, ok := ForEventReminder(base, due.Add(2*time.Hour)); ok {
			t.Error("a reminder arrived long after it could help")
		}
	})

	t.Run("somebody who turned these off hears nothing", func(t *testing.T) {
		c := base
		c.Prefs.EventReminders = false
		if _, ok := ForEventReminder(c, due); ok {
			t.Error("a preference was ignored")
		}
	})

	t.Run("the key moves when the event does", func(t *testing.T) {
		// Otherwise moving an event you have already been reminded about
		// means never hearing about the new time.
		first, _ := ForEventReminder(base, due)
		moved := base
		moved.StartTime = "20:30"
		second, ok := ForEventReminder(moved, due.Add(time.Hour))
		if !ok {
			t.Fatal("the moved event produced no reminder")
		}
		if first.Key == second.Key {
			t.Error("both times share a key, so only one of them could ever be sent")
		}
	})

	t.Run("the day before says tomorrow", func(t *testing.T) {
		c := base
		c.Reminder = "1 day before"
		// 08:00 Lagos on the 24th is 07:00 UTC.
		n, ok := ForEventReminder(c, time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC))
		if !ok {
			t.Fatal("no reminder the day before")
		}
		if !strings.Contains(n.Message.Body, "Tomorrow") {
			t.Errorf("body = %q, want it to say tomorrow", n.Message.Body)
		}
	})

	t.Run("an all-day event says the day and no time", func(t *testing.T) {
		c := base
		c.StartTime = ""
		c.Reminder = "the morning of"
		n, ok := ForEventReminder(c, time.Date(2026, 9, 25, 7, 0, 0, 0, time.UTC))
		if !ok {
			t.Fatal("no reminder for an all-day event")
		}
		if strings.Contains(n.Message.Body, "at ") {
			t.Errorf("body = %q, there is no time to give", n.Message.Body)
		}
	})
}

func TestOccursOn(t *testing.T) {
	leapDay := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	day := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	t.Run("the same day of any year", func(t *testing.T) {
		wedding := day(2019, time.September, 30)
		if !OccursOn(wedding, day(2026, time.September, 30)) {
			t.Error("an anniversary did not come round")
		}
		if OccursOn(wedding, day(2026, time.September, 29)) {
			t.Error("it came round a day early")
		}
	})

	t.Run("the twenty-ninth of February falls back to the twenty-eighth", func(t *testing.T) {
		// 2026 has no 29th. Skipping it would mean three years in four with
		// no anniversary at all.
		if !OccursOn(leapDay, day(2026, time.February, 28)) {
			t.Error("a leap-day anniversary was skipped in a common year")
		}
		if OccursOn(leapDay, day(2026, time.March, 1)) {
			t.Error("it landed on the wrong day")
		}
	})

	t.Run("and stays put in a leap year", func(t *testing.T) {
		if !OccursOn(leapDay, day(2028, time.February, 29)) {
			t.Error("a leap-day anniversary moved in a leap year")
		}
		if OccursOn(leapDay, day(2028, time.February, 28)) {
			t.Error("it came round twice")
		}
	})

	t.Run("2100 is not a leap year", func(t *testing.T) {
		if !OccursOn(leapDay, day(2100, time.February, 28)) {
			t.Error("the century rule was missed")
		}
	})
}

func TestForImportantDates(t *testing.T) {
	if _, err := time.LoadLocation("Africa/Lagos"); err != nil {
		t.Skip("no timezone database here")
	}
	base := ImportantDateCandidate{
		UserID: uuid.New(), MilestoneID: uuid.New(),
		Title:    "Our wedding",
		Timezone: "Africa/Lagos",
		Date:     time.Date(2019, 9, 30, 0, 0, 0, 0, time.UTC),
		Reminder: true,
		Prefs:    Preferences{ImportantDates: true},
	}
	// 08:00 Lagos is 07:00 UTC.
	morningOfTheDay := time.Date(2026, 9, 30, 7, 0, 0, 0, time.UTC)
	weekBefore := time.Date(2026, 9, 23, 7, 0, 0, 0, time.UTC)

	t.Run("a week's notice, so there is time to do something", func(t *testing.T) {
		due := ForImportantDates(base, weekBefore)
		if len(due) != 1 {
			t.Fatalf("got %d notifications, want one", len(due))
		}
		if !strings.Contains(due[0].Message.Body, "30 September") {
			t.Errorf("body = %q, should name the day", due[0].Message.Body)
		}
		if due[0].Message.Title != "Our wedding" {
			t.Errorf("title = %q, should name the date", due[0].Message.Title)
		}
	})

	t.Run("and the morning itself, counting the years", func(t *testing.T) {
		due := ForImportantDates(base, morningOfTheDay)
		if len(due) != 1 {
			t.Fatalf("got %d notifications, want one", len(due))
		}
		if !strings.Contains(due[0].Message.Body, "7 years today") {
			t.Errorf("body = %q, want the years counted", due[0].Message.Body)
		}
	})

	t.Run("the two are claimed apart", func(t *testing.T) {
		// One key for both would mean the week's notice silenced the day.
		week := ForImportantDates(base, weekBefore)
		day := ForImportantDates(base, morningOfTheDay)
		if week[0].Key == day[0].Key {
			t.Error("both leads share a key, so only one could ever be sent")
		}
	})

	t.Run("nothing in the small hours", func(t *testing.T) {
		// 02:00 Lagos on the day itself.
		if due := ForImportantDates(base, time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)); len(due) != 0 {
			t.Errorf("got %d notifications before anyone was awake", len(due))
		}
	})

	t.Run("but a late tick the same day still catches it", func(t *testing.T) {
		// 22:00 Lagos. An anniversary missed is missed for a year.
		if due := ForImportantDates(base, time.Date(2026, 9, 30, 21, 0, 0, 0, time.UTC)); len(due) != 1 {
			t.Errorf("got %d notifications, want the day's one", len(due))
		}
	})

	t.Run("an ordinary day says nothing", func(t *testing.T) {
		if due := ForImportantDates(base, time.Date(2026, 6, 14, 7, 0, 0, 0, time.UTC)); len(due) != 0 {
			t.Errorf("got %d notifications on a day that is not the day", len(due))
		}
	})

	t.Run("a date kept but not celebrated is never announced", func(t *testing.T) {
		c := base
		c.Reminder = false
		if due := ForImportantDates(c, morningOfTheDay); len(due) != 0 {
			t.Error("a date with its reminder off was announced")
		}
	})

	t.Run("nor is one somebody has switched off", func(t *testing.T) {
		c := base
		c.Prefs.ImportantDates = false
		if due := ForImportantDates(c, morningOfTheDay); len(due) != 0 {
			t.Error("a preference was ignored")
		}
	})

	t.Run("the first year is a day, not an anniversary", func(t *testing.T) {
		c := base
		c.Date = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
		due := ForImportantDates(c, morningOfTheDay)
		if len(due) != 1 {
			t.Fatalf("got %d notifications, want one", len(due))
		}
		if strings.Contains(due[0].Message.Body, "0 years") {
			t.Errorf("body = %q, nobody says nought years", due[0].Message.Body)
		}
	})

	t.Run("one year reads as one, not 1", func(t *testing.T) {
		c := base
		c.Date = time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
		due := ForImportantDates(c, morningOfTheDay)
		if !strings.Contains(due[0].Message.Body, "One year today") {
			t.Errorf("body = %q", due[0].Message.Body)
		}
	})
}

func TestForWritten(t *testing.T) {
	author, partner := uuid.New(), uuid.New()
	item := uuid.New()
	sent := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	note := WrittenCandidate{
		UserID: partner, AuthorID: author, AuthorName: "Ada",
		ItemID: item, Kind: KindAppreciation, WrittenAt: sent,
		Settles: appreciationUndoWindow,
		Prefs:   Preferences{Appreciation: true, Journal: true},
	}

	t.Run("the partner hears about a note, once it can no longer be taken back", func(t *testing.T) {
		n, ok := ForWritten(note, sent.Add(appreciationUndoWindow+time.Second))
		if !ok {
			t.Fatal("nobody was told about an appreciation")
		}
		if !strings.Contains(n.Message.Title, "Ada") {
			t.Errorf("title = %q, should say who", n.Message.Title)
		}
		if n.Message.Path != "/together/appreciation" {
			t.Errorf("path = %q", n.Message.Path)
		}
		if n.Key != item.String() {
			t.Errorf("key = %q, want the note's own id", n.Key)
		}
	})

	t.Run("not while it can still be undone", func(t *testing.T) {
		// A note taken back ten seconds later should never have reached a
		// lock screen, and this is the only place that can promise it.
		if _, ok := ForWritten(note, sent.Add(10*time.Second)); ok {
			t.Error("an appreciation was announced inside its undo window")
		}
	})

	t.Run("the sender never hears about their own", func(t *testing.T) {
		c := note
		c.UserID = author
		if _, ok := ForWritten(c, sent.Add(time.Minute)); ok {
			t.Error("somebody was told about something they wrote themselves")
		}
	})

	t.Run("nor does a day-old backlog go out", func(t *testing.T) {
		if _, ok := ForWritten(note, sent.Add(25*time.Hour)); ok {
			t.Error("a note from yesterday was announced today")
		}
	})

	t.Run("a journal entry goes out at once, and reads differently", func(t *testing.T) {
		c := note
		c.Kind, c.Settles = KindJournal, 0
		n, ok := ForWritten(c, sent.Add(time.Second))
		if !ok {
			t.Fatal("nobody was told about a journal entry")
		}
		if n.Message.Path != "/together/journal" {
			t.Errorf("path = %q", n.Message.Path)
		}
		if strings.Contains(n.Message.Title, "appreciated") {
			t.Errorf("title = %q, that is the other one", n.Message.Title)
		}
	})

	t.Run("each switch is read on its own", func(t *testing.T) {
		// Turning off journal notifications must not silence appreciations.
		c := note
		c.Prefs.Journal = false
		if _, ok := ForWritten(c, sent.Add(time.Minute)); !ok {
			t.Error("the journal switch silenced an appreciation")
		}
		c = note
		c.Prefs.Appreciation = false
		if _, ok := ForWritten(c, sent.Add(time.Minute)); ok {
			t.Error("the appreciation switch was ignored")
		}
	})

	t.Run("nothing about the note itself goes out", func(t *testing.T) {
		// FR-NOTF-005: what one of them wrote is the one thing a lock screen
		// must not carry.
		n, _ := ForWritten(note, sent.Add(time.Minute))
		if strings.Contains(n.Message.Body, "appreciate") || len(n.Message.Body) > 60 {
			t.Errorf("body = %q, want it to say nothing of the words", n.Message.Body)
		}
	})
}

func TestForWritten_Goals(t *testing.T) {
	author, partner := uuid.New(), uuid.New()
	sent := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	base := WrittenCandidate{
		UserID: partner, AuthorID: author, AuthorName: "Ada",
		ItemID: uuid.New(), Subject: "A place of our own",
		Kind: KindGoal, WrittenAt: sent,
		Prefs: Preferences{Goals: true},
	}

	t.Run("the partner hears that something went in", func(t *testing.T) {
		n, ok := ForWritten(base, sent.Add(time.Minute))
		if !ok {
			t.Fatal("nobody was told about progress on a goal")
		}
		if !strings.Contains(n.Message.Title, "A place of our own") {
			t.Errorf("title = %q, should name the goal", n.Message.Title)
		}
		if n.Message.Path != "/together/goals" {
			t.Errorf("path = %q", n.Message.Path)
		}
	})

	t.Run("but never how much", func(t *testing.T) {
		// A goal is a shared plan and may be named (FR-NOTF-005.AC2). What
		// somebody just moved into their savings is not for a lock screen.
		n, _ := ForWritten(base, sent.Add(time.Minute))
		if strings.ContainsAny(n.Message.Title+n.Message.Body, "0123456789₦") {
			t.Errorf("an amount reached the lock screen: %q / %q", n.Message.Title, n.Message.Body)
		}
	})

	t.Run("goals are off unless asked for", func(t *testing.T) {
		c := base
		c.Prefs.Goals = false
		if _, ok := ForWritten(c, sent.Add(time.Minute)); ok {
			t.Error("a goal notification went out with the switch off")
		}
	})

	t.Run("and never to whoever logged it", func(t *testing.T) {
		c := base
		c.UserID = author
		if _, ok := ForWritten(c, sent.Add(time.Minute)); ok {
			t.Error("somebody was told about their own entry")
		}
	})
}

func TestForChallenge(t *testing.T) {
	if _, err := time.LoadLocation("Africa/Lagos"); err != nil {
		t.Skip("no timezone database here")
	}
	base := ChallengeCandidate{
		UserID: uuid.New(), ChallengeID: uuid.New(),
		Title: "Seven days of noticing", Timezone: "Africa/Lagos",
		Day: 3, Days: 7,
		Prefs: Preferences{Challenges: true},
	}
	// 08:00 Lagos is 07:00 UTC.
	morning := time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC)

	t.Run("a day not yet marked is nudged in the morning", func(t *testing.T) {
		n, ok := ForChallenge(base, morning)
		if !ok {
			t.Fatal("no nudge for an unmarked day")
		}
		if !strings.Contains(n.Message.Body, "Day 3 of 7") {
			t.Errorf("body = %q, should say where they are", n.Message.Body)
		}
	})

	t.Run("not before anyone is awake", func(t *testing.T) {
		if _, ok := ForChallenge(base, time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)); ok {
			t.Error("a nudge went out in the small hours")
		}
	})

	t.Run("and not once they have marked it", func(t *testing.T) {
		c := base
		c.MarkedToday = true
		if _, ok := ForChallenge(c, morning); ok {
			t.Error("somebody was nudged about a day they had already marked")
		}
	})

	t.Run("one a day, keyed on their own date", func(t *testing.T) {
		first, _ := ForChallenge(base, morning)
		later, _ := ForChallenge(base, morning.Add(6*time.Hour))
		if first.Key != later.Key {
			t.Error("two ticks the same day would send two nudges")
		}
		next, _ := ForChallenge(base, morning.Add(24*time.Hour))
		if next.Key == first.Key {
			t.Error("tomorrow shares today's key, so tomorrow would be silent")
		}
	})

	t.Run("nothing before it starts or after it ends", func(t *testing.T) {
		for _, day := range []int{0, 8} {
			c := base
			c.Day = day
			if _, ok := ForChallenge(c, morning); ok {
				t.Errorf("a nudge went out on day %d of 7", day)
			}
		}
	})

	t.Run("challenges are off unless asked for", func(t *testing.T) {
		c := base
		c.Prefs.Challenges = false
		if _, ok := ForChallenge(c, morning); ok {
			t.Error("a nudge went out with the switch off")
		}
	})

	t.Run("it never counts what was missed", func(t *testing.T) {
		// A challenge is not a streak (DEC-30): a skipped day is a day, not
		// a failure, and nothing here should imply otherwise.
		n, _ := ForChallenge(base, morning)
		for _, word := range []string{"miss", "streak", "behind", "broke"} {
			if strings.Contains(strings.ToLower(n.Message.Body+n.Message.Title), word) {
				t.Errorf("body = %q, want no scolding", n.Message.Body)
			}
		}
	})
}
