package notifications

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

type nudgeRepo struct {
	*fakeRepo
	partner uuid.UUID
	name    string
	counts  map[string]int
	// nudgesOff is the partner having switched nudges off.
	nudgesOff bool
}

func (r *nudgeRepo) NudgeTarget(context.Context, uuid.UUID) (uuid.UUID, string, error) {
	return r.partner, r.name, nil
}
func (r *nudgeRepo) CountSends(_ context.Context, _ uuid.UUID, kind string, _ time.Time) (int, error) {
	return r.counts[kind], nil
}
func (r *nudgeRepo) PreferencesFor(context.Context, uuid.UUID) (Preferences, bool, error) {
	p := Defaults()
	p.Nudges = !r.nudgesOff
	return p, true, nil
}
func (r *nudgeRepo) SavePreferences(context.Context, uuid.UUID, Preferences, time.Time) error {
	return nil
}
func (r *nudgeRepo) Subscribe(context.Context, Subscription, time.Time) error { return nil }

func newNudgeService(t *testing.T, sender push.Sender) (*Service, *nudgeRepo) {
	t.Helper()
	base := newFakeRepo()
	base.subs = []Subscription{{Endpoint: "https://push.test/a", P256dh: "k", Auth: "a"}}
	base.budget = Budget{Prefs: Preferences{DailyCap: 6}, Timezone: "UTC"}
	repo := &nudgeRepo{fakeRepo: base, partner: uuid.New(), name: "Falola", counts: map[string]int{}}
	svc := NewService(repo, sender, slog.New(slog.DiscardHandler), func() time.Time { return at("12:00") })
	return svc, repo
}

func TestNudge(t *testing.T) {
	ctx := context.Background()

	t.Run("switched off, the sender is told so and nothing is sent", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.nudgesOff = true
		if _, err := svc.Nudge(ctx, uuid.New()); !errors.Is(err, ErrNudgesOff) {
			t.Fatalf("err = %v, want ErrNudgesOff", err)
		}
		if len(sender.sent) != 0 {
			t.Errorf("%d sent, want none", len(sender.sent))
		}
	})

	t.Run("it reaches the other one, and says who from", func(t *testing.T) {
		sender := &fakeSender{}
		svc, _ := newNudgeService(t, sender)
		left, err := svc.Nudge(ctx, uuid.New())
		if err != nil {
			t.Fatalf("nudge: %v", err)
		}
		if left != nudgesPerDay-1 {
			t.Errorf("left = %d, want %d", left, nudgesPerDay-1)
		}
		if len(sender.sent) != 1 {
			t.Fatalf("%d sent, want 1", len(sender.sent))
		}
		if sender.sent[0].Title != "From Falola" {
			t.Errorf("title = %q", sender.sent[0].Title)
		}
		// There is nothing to write and nothing to answer: that is the point.
		if sender.sent[0].Body != "Thinking about you." {
			t.Errorf("body = %q", sender.sent[0].Body)
		}
	})

	t.Run("three a day, and then tomorrow", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.counts[KindNudge] = nudgesPerDay
		if _, err := svc.Nudge(ctx, uuid.New()); !errors.Is(err, ErrNudgesSpent) {
			t.Errorf("err = %v, want the day's allowance to be spent", err)
		}
		if len(sender.sent) != 0 {
			t.Error("it was sent anyway")
		}
	})

	// Being told they are asleep is a better answer than waking them, and a
	// far better one than silence that looks like it worked.
	t.Run("not while they are asleep", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.fakeRepo.budget = Budget{
			Prefs:    Preferences{QuietFrom: "22:00", QuietTo: "07:00", DailyCap: 6},
			Timezone: "UTC",
		}
		svc.now = func() time.Time { return at("23:30") }
		if _, err := svc.Nudge(ctx, uuid.New()); !errors.Is(err, ErrTheyAreResting) {
			t.Errorf("err = %v, want their quiet hours", err)
		}
		if len(sender.sent) != 0 {
			t.Error("it woke them")
		}
	})

	t.Run("not once their day is full", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.fakeRepo.budget = Budget{Prefs: Preferences{DailyCap: 2}, Timezone: "UTC", SentToday: 2}
		if _, err := svc.Nudge(ctx, uuid.New()); !errors.Is(err, ErrTheyHaveHadEnough) {
			t.Errorf("err = %v, want their cap", err)
		}
	})

	t.Run("nobody to nudge yet", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.partner = uuid.Nil
		if _, err := svc.Nudge(ctx, uuid.New()); !errors.Is(err, ErrNoPartner) {
			t.Errorf("err = %v, want no partner", err)
		}
	})

	// A browser that unsubscribed is not the sender's problem: the thought
	// was sent either way.
	t.Run("nothing listening is still a nudge sent", func(t *testing.T) {
		sender := &fakeSender{}
		svc, repo := newNudgeService(t, sender)
		repo.fakeRepo.subs = nil
		if _, err := svc.Nudge(ctx, uuid.New()); err != nil {
			t.Errorf("err = %v, want no complaint", err)
		}
	})
}
