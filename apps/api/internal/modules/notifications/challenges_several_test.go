package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func running(user uuid.UUID, title string, day int) ChallengeCandidate {
	return ChallengeCandidate{
		UserID: user, ChallengeID: uuid.New(), Title: title, Timezone: "UTC",
		Day: day, Days: 7, Prefs: Preferences{Challenges: true},
	}
}

func TestForChallenges_OneKeepsItsOwnMessageTwoOrMoreAreCombined(t *testing.T) {
	me := uuid.New()
	morning := at("09:00")

	t.Run("one is the single-challenge message", func(t *testing.T) {
		got := ForChallenges([]ChallengeCandidate{running(me, "Ten conversations", 3)}, morning)
		if len(got) != 1 || got[0].Message.Title != "Ten conversations" || got[0].Message.Body != "Day 3 of 7 is waiting for you." {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("two or more are one push naming them", func(t *testing.T) {
		got := ForChallenges([]ChallengeCandidate{
			running(me, "Seven days of gratitude", 1),
			running(me, "Ten conversations", 4),
		}, morning)
		if len(got) != 1 {
			t.Fatalf("pushes = %d, want 1", len(got))
		}
		m := got[0].Message
		if m.Title != "2 challenges today" || m.Body != "Seven days of gratitude · Ten conversations" || m.Path != "/together/challenges" {
			t.Errorf("message = %+v", m)
		}
		if got[0].Kind != KindChallenge || got[0].UserID != me {
			t.Errorf("notification = %+v", got[0])
		}

		three := ForChallenges([]ChallengeCandidate{
			running(me, "A", 1), running(me, "B", 1), running(me, "C", 1),
		}, morning)
		if len(three) != 1 || three[0].Message.Title != "3 challenges today" || three[0].Message.Body != "A · B · C" {
			t.Errorf("three = %+v", three)
		}
	})

	t.Run("only the ones still waiting count", func(t *testing.T) {
		marked := running(me, "Marked", 2)
		marked.MarkedToday = true
		over := running(me, "Over", 9) // past its last day
		got := ForChallenges([]ChallengeCandidate{marked, running(me, "Waiting", 2), over}, morning)
		if len(got) != 1 || got[0].Message.Title != "Waiting" {
			t.Fatalf("got %+v, want just the one still waiting, in its own words", got)
		}
		if got := ForChallenges([]ChallengeCandidate{marked, over}, morning); len(got) != 0 {
			t.Errorf("nothing waiting still sent %+v", got)
		}
	})

	t.Run("each person gets their own", func(t *testing.T) {
		you := uuid.New()
		got := ForChallenges([]ChallengeCandidate{
			running(me, "A", 1), running(you, "A", 1), running(me, "B", 1),
		}, morning)
		if len(got) != 2 {
			t.Fatalf("pushes = %d, want one each", len(got))
		}
		for _, n := range got {
			switch n.UserID {
			case me:
				if n.Message.Title != "2 challenges today" {
					t.Errorf("mine = %+v", n.Message)
				}
			case you:
				if n.Message.Title != "A" {
					t.Errorf("theirs = %+v", n.Message)
				}
			}
		}
	})
}

// However many are running, a person's key for the day is one key, so the
// claim lets only the first push through.
func TestForChallenges_OnePerPersonPerDay(t *testing.T) {
	me := uuid.New()
	cs := []ChallengeCandidate{running(me, "A", 1), running(me, "B", 1)}
	first := ForChallenges(cs, at("09:00"))
	later := ForChallenges(cs, at("15:00"))
	single := ForChallenges(cs[:1], at("09:00"))
	if first[0].Key != later[0].Key || first[0].Key != single[0].Key {
		t.Errorf("keys differ within a day: %q, %q, %q", first[0].Key, later[0].Key, single[0].Key)
	}
	if next := ForChallenges(cs, at("09:00").Add(24*time.Hour)); next[0].Key == first[0].Key {
		t.Error("tomorrow shares today's key, so tomorrow would be silent")
	}

	repo := newFakeRepo()
	repo.budget = Budget{Timezone: "UTC"}
	repo.subs = []Subscription{{Endpoint: "https://push.test/a", P256dh: "k", Auth: "a"}}
	repo.challenges = cs
	sender := &fakeSender{}
	worker := NewWorker(repo, sender, func() time.Time { return at("09:00") }, quietLog())
	if sent, err := worker.Tick(t.Context()); err != nil || sent != 1 {
		t.Fatalf("first tick sent %d (err %v), want one combined push", sent, err)
	}
	if sent, err := worker.Tick(t.Context()); err != nil || sent != 0 {
		t.Fatalf("second tick sent %d (err %v), want none: already told today", sent, err)
	}
	if len(sender.sent) != 1 || sender.sent[0].Title != "2 challenges today" {
		t.Errorf("delivered = %+v", sender.sent)
	}
}
