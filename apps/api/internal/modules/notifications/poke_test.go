package notifications

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// blockingRepo blocks its first CurrentWeekCandidates call until released,
// so a test can hold one pass open while more pokes arrive; every call
// after that returns at once, so a coalesced extra pass finishes quickly.
type blockingRepo struct {
	*fakeRepo
	mu      sync.Mutex
	calls   int
	entered bool
	enter   chan struct{}
	release chan struct{}
}

func newBlockingRepo() *blockingRepo {
	return &blockingRepo{
		fakeRepo: newFakeRepo(),
		enter:    make(chan struct{}),
		release:  make(chan struct{}),
	}
}

func (r *blockingRepo) CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error) {
	r.mu.Lock()
	r.calls++
	first := !r.entered
	r.entered = true
	r.mu.Unlock()

	if first {
		close(r.enter)
		<-r.release
	}
	return nil, nil
}

func (r *blockingRepo) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func TestWorkerPoke_RunsOnePass(t *testing.T) {
	repo := newFakeRepo()
	w := workerFor(repo, &fakeSender{})

	done := make(chan struct{})
	w.afterPoke = func() { close(done) }

	w.Poke()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poke never ran a pass")
	}
	if got := len(repo.candidates); got != 0 {
		t.Fatalf("setup: candidates should be empty, got %d", got)
	}
}

// TestWorkerPoke_CoalescesConcurrentPokes is the coalescing guarantee: two
// pokes that land while a pass is already running must produce exactly one
// extra pass afterwards, never two, and never zero.
func TestWorkerPoke_CoalescesConcurrentPokes(t *testing.T) {
	repo := newBlockingRepo()
	w := workerFor(repo, &fakeSender{})

	passes := make(chan struct{}, 8)
	w.afterPoke = func() { passes <- struct{}{} }

	w.Poke()
	<-repo.enter // the first pass is now in flight, holding the lock

	// Two more pokes arrive while that pass is still running.
	w.Poke()
	w.Poke()

	close(repo.release) // let the first pass finish

	select {
	case <-passes:
	case <-time.After(2 * time.Second):
		t.Fatal("the first pass never finished")
	}

	// Exactly one coalesced pass should follow it.
	select {
	case <-passes:
	case <-time.After(2 * time.Second):
		t.Fatal("no extra pass ran for the pokes that arrived mid-run")
	}

	select {
	case <-passes:
		t.Fatal("a second extra pass ran; the pokes should have coalesced into one")
	case <-time.After(200 * time.Millisecond):
	}

	if got := repo.callCount(); got != 2 {
		t.Errorf("calls = %d, want exactly 2 (the run plus one coalesced extra)", got)
	}
}

func TestWorkerPoke_NeverOverlapsACronTick(t *testing.T) {
	repo := newBlockingRepo()
	w := workerFor(repo, &fakeSender{})

	tickDone := make(chan struct{})
	go func() {
		_, _, _ = w.TryTick(context.Background())
		close(tickDone)
	}()
	<-repo.enter // the cron-style tick now holds the lock

	// A second caller — standing in for a poke — must be told nothing ran,
	// not be let in alongside it.
	if _, ran, err := w.TryTick(context.Background()); ran || err != nil {
		t.Fatalf("ran = %v, err = %v, want ran=false while the tick above is in flight", ran, err)
	}

	close(repo.release)
	<-tickDone
}

func TestPoker_IsSatisfiedByWorker(t *testing.T) {
	var _ Poker = (*Worker)(nil)
}

func TestNotificationsRecorded_InInbox(t *testing.T) {
	setter := uuid.New()
	week := uuid.New()
	candidate := Candidate{
		UserID: setter, SetterUserID: setter, WeekID: week, WeekStatus: "draft",
		Prefs: Preferences{NewWeek: true},
	}
	repo := newFakeRepo()
	repo.candidates = []Candidate{candidate}
	repo.subs = []Subscription{{Endpoint: "https://push.example/abc", P256dh: "k", Auth: "a"}}
	w := workerFor(repo, &fakeSender{})

	if _, err := w.Tick(context.Background()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(repo.inbox) != 1 {
		t.Fatalf("inbox rows = %d, want 1", len(repo.inbox))
	}
	if repo.inbox[0].UserID != setter || repo.inbox[0].Kind != KindNewWeek {
		t.Errorf("inbox row = %+v, want the setter's new-week notification", repo.inbox[0])
	}
}

func TestNotificationsRecorded_EvenWithNoDevice(t *testing.T) {
	// Inbox history and a lock-screen push are separate things: RecordInbox
	// records a decided send regardless of whether anyone was subscribed.
	setter := uuid.New()
	candidate := Candidate{
		UserID: setter, SetterUserID: setter, WeekID: uuid.New(), WeekStatus: "draft",
		Prefs: Preferences{NewWeek: true},
	}
	repo := newFakeRepo()
	repo.candidates = []Candidate{candidate}
	w := workerFor(repo, &fakeSender{})

	if _, err := w.Tick(context.Background()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(repo.inbox) != 1 {
		t.Fatalf("inbox rows = %d, want 1 even with nobody subscribed", len(repo.inbox))
	}
}

func TestWakeWhenSettled(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	me, them := uuid.New(), uuid.New()
	written := func(ago, settles time.Duration) WrittenCandidate {
		return WrittenCandidate{UserID: me, AuthorID: them, WrittenAt: now.Add(-ago), Settles: settles}
	}
	booked := func(w *Worker) time.Time {
		w.pokeMu.Lock()
		defer w.pokeMu.Unlock()
		if w.wake != nil {
			w.wake.Stop()
		}
		return w.wakeFor
	}

	t.Run("a pass is booked for when the earliest window closes", func(t *testing.T) {
		w := NewWorker(nil, nil, func() time.Time { return now }, nil)
		w.wakeWhenSettled([]WrittenCandidate{
			written(10*time.Second, time.Minute),
			written(20*time.Second, 30*time.Second), // closes at now+10s: the earliest
		}, now)
		if got, want := booked(w), now.Add(10*time.Second); !got.Equal(want) {
			t.Errorf("booked for %v, want %v", got, want)
		}
	})

	t.Run("nothing is booked when nothing is waiting", func(t *testing.T) {
		w := NewWorker(nil, nil, func() time.Time { return now }, nil)
		w.wakeWhenSettled([]WrittenCandidate{
			written(time.Minute, 30*time.Second), // already settled
			written(0, 0),                        // a journal entry: no window at all
			{UserID: them, AuthorID: them, WrittenAt: now, Settles: time.Minute}, // your own
		}, now)
		if got := booked(w); !got.IsZero() {
			t.Errorf("booked for %v, want nothing", got)
		}
	})

	t.Run("a window further off than a few minutes is the cron's", func(t *testing.T) {
		w := NewWorker(nil, nil, func() time.Time { return now }, nil)
		w.wakeWhenSettled([]WrittenCandidate{written(0, time.Hour)}, now)
		if got := booked(w); !got.IsZero() {
			t.Errorf("booked for %v, want nothing", got)
		}
	})
}
