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
		UserID: uuid.New(), EventID: id,
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
		// FR-NOTF-005: a lock screen never carries what they wrote.
		if strings.Contains(n.Message.Body+n.Message.Title, "Dinner") {
			t.Errorf("the event's own words reached the lock screen: %q", n.Message.Body)
		}
		if !strings.Contains(n.Message.Body, "7:30 pm") {
			t.Errorf("body = %q, should say when it is", n.Message.Body)
		}
		if !strings.Contains(n.Message.Body, "Today") {
			t.Errorf("body = %q, should place the day", n.Message.Body)
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
