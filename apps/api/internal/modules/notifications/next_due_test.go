package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNextDueFrom(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	safety := now.Add(safetyInterval)
	utcDay := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	remind := Preferences{EventReminders: true}

	tests := []struct {
		name string
		seen seen
		want time.Time
	}{
		{
			name: "nothing seen waits out the safety interval",
			want: safety,
		},
		{
			name: "the soonest event reminder wins",
			seen: seen{events: []EventCandidate{
				{Timezone: "UTC", Date: utcDay, StartTime: "13:00", Reminder: "an hour before", Prefs: remind},    // 12:00
				{Timezone: "UTC", Date: utcDay, StartTime: "11:30", Reminder: "30 minutes before", Prefs: remind}, // 11:00
			}},
			want: time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC),
		},
		{
			// UTC-11: its midnight is 11:00 here, its morning long after.
			name: "an anniversary wakes the worker at the couple's midnight",
			seen: seen{dates: []ImportantDateCandidate{
				{Timezone: "Etc/GMT+11", Anniversary: true, Reminder: true, Prefs: Preferences{ImportantDates: true}},
			}},
			want: time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC),
		},
		{
			name: "a reminder already gone by is not waited for",
			seen: seen{events: []EventCandidate{
				{Timezone: "UTC", Date: utcDay, StartTime: "10:30", Reminder: "an hour before", Prefs: remind}, // 09:30
			}},
			want: safety,
		},
		{
			name: "event reminders switched off are not waited for",
			seen: seen{events: []EventCandidate{
				{Timezone: "UTC", Date: utcDay, StartTime: "11:30", Reminder: "30 minutes before"},
			}},
			want: safety,
		},
		{
			name: "a reminder further off than the safety interval is the safety interval's to find",
			seen: seen{events: []EventCandidate{
				{Timezone: "UTC", Date: utcDay, StartTime: "20:00", Reminder: "at the time", Prefs: remind},
			}},
			want: safety,
		},
		{
			name: "an event ending soon is waited for, to ask how it went",
			seen: seen{ended: []EventCandidate{
				{Timezone: "UTC", Date: utcDay, StartTime: "09:00", EndTime: "12:00", Prefs: Preferences{EventFollowups: true}},
			}},
			want: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "today's prayer reminder, when still ahead",
			seen: seen{people: []Candidate{
				{Timezone: "UTC", Prefs: Preferences{PrayerReminder: true, ReminderTime: "11:15"}},
			}},
			want: time.Date(2026, 9, 23, 11, 15, 0, 0, time.UTC),
		},
		{
			name: "a prayer reminder already passed today is tomorrow's, not a moment in the past",
			seen: seen{people: []Candidate{
				{Timezone: "UTC", Prefs: Preferences{PrayerReminder: true, ReminderTime: "09:00"}},
			}},
			want: safety, // tomorrow's is 23 hours off
		},
		{
			name: "a prayer reminder in the person's own zone",
			seen: seen{people: []Candidate{
				// 10:00 UTC is 06:00 in New York; their 07:30 is 11:30 UTC.
				{Timezone: "America/New_York", Prefs: Preferences{PrayerReminder: true, ReminderTime: "07:30"}},
			}},
			want: time.Date(2026, 9, 23, 11, 30, 0, 0, time.UTC),
		},
		{
			name: "a prayer reminder switched off is not waited for",
			seen: seen{people: []Candidate{
				{Timezone: "UTC", Prefs: Preferences{ReminderTime: "11:15"}},
			}},
			want: safety,
		},
		{
			name: "the morning of each couple's zone, the nearest first",
			seen: seen{dates: []ImportantDateCandidate{
				{Timezone: "Asia/Dubai"},       // 14:00 there; tomorrow's 08:00 is 04:00 UTC
				{Timezone: "America/New_York"}, // 06:00 there; today's 08:00 is 12:00 UTC
				{Timezone: "UTC"},              // already past 08:00
			}},
			want: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "challenges and memories keep their own zone's morning",
			seen: seen{
				challenges: []ChallengeCandidate{{Timezone: "America/New_York"}},
				memories:   []MemoryAnniversaryCandidate{{Timezone: "Europe/London"}},
			},
			want: time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "a notification held back is waited for until it can go",
			seen: seen{held: []time.Time{time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC)}},
			want: time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC),
		},
		{
			name: "an unknown zone is read as UTC, as everywhere else",
			seen: seen{dates: []ImportantDateCandidate{{Timezone: "Nowhere/Land"}}},
			want: safety, // UTC's morning has passed today
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextDueFrom(now, tt.seen); !got.Equal(tt.want) {
				t.Errorf("nextDueFrom = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNextClock_TomorrowWhenTodaysHasPassed(t *testing.T) {
	now := time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)

	if got, want := nextClock(now, time.UTC, 7*60), time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("nextClock = %s, want %s", got, want)
	}
	// Exactly now counts as passed, matching ReminderPassed.
	if got, want := nextClock(now, time.UTC, 20*60), time.Date(2026, 9, 24, 20, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("nextClock at exactly now = %s, want %s", got, want)
	}
	// Read in the zone's own day: 20:00 UTC is already 05:00 tomorrow in Tokyo,
	// so its 07:00 is two hours away.
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("loading Tokyo: %v", err)
	}
	if got, want := nextClock(now, tokyo, 7*60), time.Date(2026, 9, 23, 22, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("nextClock in Tokyo = %s, want %s", got, want)
	}
}

func TestHeldUntil(t *testing.T) {
	now := time.Date(2026, 9, 23, 23, 0, 0, 0, time.UTC)
	b := Budget{Prefs: Preferences{QuietFrom: "22:00", QuietTo: "07:00"}, Timezone: "UTC"}

	if at, ok := heldUntil("quiet_hours", true, b, now); !ok || !at.Equal(time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC)) {
		t.Errorf("quiet hours: %s, %v", at, ok)
	}
	if at, ok := heldUntil("daily_cap", true, b, now); !ok || !at.Equal(time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("daily cap: %s, %v", at, ok)
	}
	if _, ok := heldUntil("quiet_hours", false, b, now); ok {
		t.Error("a perishable notification is dropped, so nothing waits on it")
	}
	if _, ok := heldUntil("something_else", true, b, now); ok {
		t.Error("an unknown reason was waited for")
	}
}

// countingRepo counts how often a pass reaches the database.
type countingRepo struct {
	*fakeRepo
	passes  atomic.Int32
	failing atomic.Bool
}

func newCountingRepo() *countingRepo { return &countingRepo{fakeRepo: newFakeRepo()} }

func (r *countingRepo) CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error) {
	r.passes.Add(1)
	if r.failing.Load() {
		return nil, context.DeadlineExceeded
	}
	return r.fakeRepo.CurrentWeekCandidates(ctx, now)
}

// clockedWorker is a worker whose clock the test moves.
func clockedWorker(repo WorkerRepository, at *time.Time) *Worker {
	return NewWorker(repo, &fakeSender{}, func() time.Time { return *at }, quietLog())
}

func TestTickIfDue_FreshProcessRunsThenWaits(t *testing.T) {
	repo := newCountingRepo()
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)

	if _, ran, reason, err := w.TickIfDue(context.Background()); err != nil || !ran || reason != reasonFirstPass {
		t.Fatalf("fresh process: ran=%v reason=%q err=%v", ran, reason, err)
	}
	if got := repo.passes.Load(); got != 1 {
		t.Fatalf("passes = %d, want 1", got)
	}
	if want := at.Add(safetyInterval); !w.NextDue().Equal(want) {
		t.Fatalf("NextDue = %s, want %s", w.NextDue(), want)
	}

	// Five minutes later, before anything is due: no database at all.
	at = at.Add(5 * time.Minute)
	sent, ran, reason, err := w.TickIfDue(context.Background())
	if err != nil || ran || sent != 0 || reason != reasonNothing {
		t.Fatalf("early tick: sent=%d ran=%v reason=%q err=%v", sent, ran, reason, err)
	}
	if got := repo.passes.Load(); got != 1 {
		t.Errorf("an early tick reached the repository: passes = %d", got)
	}
}

func TestTickIfDue_RunsOnceNextDueArrives(t *testing.T) {
	repo := newCountingRepo()
	repo.candidates = []Candidate{{Timezone: "UTC", Prefs: Preferences{PrayerReminder: true, ReminderTime: "19:00"}}}
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)

	if _, ran, _, _ := w.TickIfDue(context.Background()); !ran {
		t.Fatal("first tick did not run")
	}
	if want := time.Date(2026, 9, 23, 19, 0, 0, 0, time.UTC); !w.NextDue().Equal(want) {
		t.Fatalf("NextDue = %s, want %s", w.NextDue(), want)
	}

	at = time.Date(2026, 9, 23, 18, 59, 0, 0, time.UTC)
	if _, ran, _, _ := w.TickIfDue(context.Background()); ran {
		t.Error("ran a minute before anything was due")
	}
	at = time.Date(2026, 9, 23, 19, 0, 0, 0, time.UTC)
	if _, ran, reason, _ := w.TickIfDue(context.Background()); !ran || reason != reasonDue {
		t.Errorf("at the due moment: ran=%v reason=%q", ran, reason)
	}
	if got := repo.passes.Load(); got != 2 {
		t.Errorf("passes = %d, want 2", got)
	}
}

func TestTickIfDue_InvalidateForcesARun(t *testing.T) {
	repo := newCountingRepo()
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)
	w.TickIfDue(context.Background())

	w.Invalidate()
	if _, ran, reason, _ := w.TickIfDue(context.Background()); !ran || reason != reasonChanged {
		t.Fatalf("after a write: ran=%v reason=%q", ran, reason)
	}
	// One write, one pass: the flag is spent.
	if _, ran, _, _ := w.TickIfDue(context.Background()); ran {
		t.Error("ran again with nothing new")
	}
	if got := repo.passes.Load(); got != 2 {
		t.Errorf("passes = %d, want 2", got)
	}
}

func TestTickIfDue_SafetyIntervalForcesARun(t *testing.T) {
	repo := newCountingRepo()
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)
	w.TickIfDue(context.Background())

	// Whatever nextDue says, the time since the last pass is a cap of its own.
	w.stateMu.Lock()
	w.nextDue = at.Add(24 * time.Hour)
	w.stateMu.Unlock()

	at = at.Add(safetyInterval - time.Minute)
	if _, ran, _, _ := w.TickIfDue(context.Background()); ran {
		t.Fatal("ran before the safety interval")
	}
	at = at.Add(time.Minute)
	if _, ran, reason, _ := w.TickIfDue(context.Background()); !ran || reason != reasonSafety {
		t.Errorf("at the safety interval: ran=%v reason=%q", ran, reason)
	}
}

func TestTickIfDue_AFailedPassIsTriedAgainAtTheNextTick(t *testing.T) {
	repo := newCountingRepo()
	repo.failing.Store(true)
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)

	if _, ran, _, err := w.TickIfDue(context.Background()); !ran || err == nil {
		t.Fatalf("expected a failed pass: ran=%v err=%v", ran, err)
	}
	repo.failing.Store(false)
	at = at.Add(5 * time.Minute)
	if _, ran, _, err := w.TickIfDue(context.Background()); !ran || err != nil {
		t.Errorf("the retry: ran=%v err=%v", ran, err)
	}
}

func TestTickIfDue_APokeStillRunsAPass(t *testing.T) {
	repo := newCountingRepo()
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	w := clockedWorker(repo, &at)
	w.TickIfDue(context.Background())

	done := make(chan struct{})
	w.afterPoke = func() { close(done) }
	w.Poke()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poke never ran a pass")
	}
	if got := repo.passes.Load(); got != 2 {
		t.Errorf("passes = %d, want 2 (the tick's and the poke's)", got)
	}
}

func TestTickIfDue_QuietHoursHoldIsWaitedFor(t *testing.T) {
	repo := newCountingRepo()
	repo.budget = Budget{Prefs: Preferences{QuietFrom: "22:00", QuietTo: "07:00"}, Timezone: "UTC"}
	at := time.Date(2026, 9, 23, 23, 0, 0, 0, time.UTC)
	repo.written = []WrittenCandidate{{
		UserID: uuid.New(), AuthorID: uuid.New(), AuthorName: "A", ItemID: uuid.New(), Kind: KindJournal,
		WrittenAt: at.Add(-time.Minute), Prefs: Preferences{Journal: true},
	}}
	w := clockedWorker(repo, &at)

	if _, ran, _, _ := w.TickIfDue(context.Background()); !ran {
		t.Fatal("first tick did not run")
	}
	if len(repo.sentTo) != 0 {
		t.Fatalf("sent during quiet hours: %v", repo.sentTo)
	}

	// The same entry, held, at a time when quiet hours end inside the cap.
	repo2 := newCountingRepo()
	repo2.budget = Budget{Prefs: Preferences{QuietFrom: "22:00", QuietTo: "07:00"}, Timezone: "UTC"}
	at2 := time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
	repo2.written = []WrittenCandidate{{
		UserID: uuid.New(), AuthorID: uuid.New(), AuthorName: "A", ItemID: uuid.New(), Kind: KindJournal,
		WrittenAt: at2.Add(-time.Minute), Prefs: Preferences{Journal: true},
	}}
	w2 := clockedWorker(repo2, &at2)
	w2.TickIfDue(context.Background())
	if want := time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC); !w2.NextDue().Equal(want) {
		t.Errorf("NextDue = %s, want the end of quiet hours %s", w2.NextDue(), want)
	}
}

func TestInvalidateOnWrite(t *testing.T) {
	tests := []struct {
		name   string
		method string
		status int // 0: the handler never sets one, which net/http sends as 200
		want   bool
	}{
		{"a successful POST", http.MethodPost, http.StatusCreated, true},
		{"a successful PATCH", http.MethodPatch, http.StatusOK, true},
		{"a successful DELETE", http.MethodDelete, http.StatusNoContent, true},
		{"a PUT that never sets a status", http.MethodPut, 0, true},
		{"a GET", http.MethodGet, http.StatusOK, false},
		{"a HEAD", http.MethodHead, http.StatusOK, false},
		{"an OPTIONS", http.MethodOptions, http.StatusNoContent, false},
		{"a POST refused for bad input", http.MethodPost, http.StatusUnprocessableEntity, false},
		{"a POST refused as unauthorised", http.MethodPost, http.StatusUnauthorized, false},
		{"a POST that failed", http.MethodPost, http.StatusInternalServerError, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
			w := clockedWorker(newFakeRepo(), &at)

			handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					rw.WriteHeader(tt.status)
				}
			})
			InvalidateOnWrite(w)(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tt.method, "/v1/x", nil))

			w.stateMu.Lock()
			got := w.dirty
			w.stateMu.Unlock()
			if got != tt.want {
				t.Errorf("dirty = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTick_NothingDueSaysWhenNext(t *testing.T) {
	repo := newCountingRepo()
	at := time.Date(2026, 9, 23, 18, 30, 0, 0, time.UTC)
	h := NewTickHandler(clockedWorker(repo, &at), testSecret, quietLog())

	if w := post(h, "Bearer "+testSecret); w.Code != http.StatusOK {
		t.Fatalf("first tick: status = %d", w.Code)
	}
	at = at.Add(5 * time.Minute)
	w := post(h, "Bearer "+testSecret)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if body["status"] != "nothing due" || body["next"] != "2026-09-23T21:30:00Z" {
		t.Errorf("body = %v", body)
	}
	if got := repo.passes.Load(); got != 1 {
		t.Errorf("the second tick reached the repository: passes = %d", got)
	}
}
