package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// ForWritten for a challenge one partner just started: the other is told, on
// one switch of its own, with a body that says when day one opens.
func TestForWritten_ChallengeStarted(t *testing.T) {
	author, partner := uuid.New(), uuid.New()
	sent := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) // a Thursday
	base := WrittenCandidate{
		UserID: partner, AuthorID: author, AuthorName: "Ada",
		ItemID: uuid.New(), Subject: "Ten conversations",
		Kind: KindChallengeStarted, WrittenAt: sent,
		Timezone: "UTC", StartsOn: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC),
		Prefs: Preferences{PartnerChallenges: true},
	}

	t.Run("a shared one that begins today", func(t *testing.T) {
		n, ok := ForWritten(base, sent.Add(time.Second))
		if !ok {
			t.Fatal("nobody was told about a challenge being started")
		}
		if n.Message.Title != "Ada started Ten conversations" || n.Message.Body != "Day 1 opens today." {
			t.Errorf("message = %+v", n.Message)
		}
		if n.Message.Path != "/together/challenges/"+base.ItemID.String() {
			t.Errorf("path = %q", n.Message.Path)
		}
		if n.Kind != KindChallengeStarted || n.Key != base.ItemID.String() {
			t.Errorf("kind = %q, key = %q; want the challenge's own id so it sends once", n.Kind, n.Key)
		}
	})

	t.Run("a scheduled one names the weekday it opens", func(t *testing.T) {
		c := base
		c.StartsOn = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC) // a Monday
		n, ok := ForWritten(c, sent.Add(time.Second))
		if !ok || n.Message.Body != "Day 1 opens Monday." {
			t.Fatalf("got %+v, %v", n.Message, ok)
		}
	})

	t.Run("today is the couple's today, not the server's", func(t *testing.T) {
		c := base
		c.Timezone = "Pacific/Auckland" // already the 25th there
		c.StartsOn = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
		n, ok := ForWritten(c, sent.Add(time.Second))
		if !ok || n.Message.Body != "Day 1 opens today." {
			t.Fatalf("got %+v, %v", n.Message, ok)
		}
	})

	t.Run("a just-me one is news, not something to do", func(t *testing.T) {
		c := base
		c.Mine = true
		n, ok := ForWritten(c, sent.Add(time.Second))
		if !ok || n.Message.Title != "Ada started Ten conversations" || n.Message.Body != "Just them — cheer them on." {
			t.Fatalf("got %+v, %v", n.Message, ok)
		}
	})

	t.Run("goes out at once", func(t *testing.T) {
		if _, ok := ForWritten(base, sent); !ok {
			t.Error("it waited as if there were something to undo")
		}
	})

	t.Run("not to whoever started it", func(t *testing.T) {
		c := base
		c.UserID = author
		if _, ok := ForWritten(c, sent.Add(time.Second)); ok {
			t.Error("the starter was told about their own challenge")
		}
	})

	t.Run("not to somebody who turned it off", func(t *testing.T) {
		c := base
		c.Prefs.PartnerChallenges = false
		if _, ok := ForWritten(c, sent.Add(time.Second)); ok {
			t.Error("told despite partner_challenges being off")
		}
	})

	t.Run("not on the daily nudge's switch", func(t *testing.T) {
		c := base
		c.Prefs = Preferences{Challenges: true, PartnerChallenges: false}
		if _, ok := ForWritten(c, sent.Add(time.Second)); ok {
			t.Error("the opt-in nudge switch let it through")
		}
	})
}

func TestPartnerChallenges_PreferenceDefaultsOnAndPatches(t *testing.T) {
	if !Defaults().PartnerChallenges {
		t.Fatal("partner_challenges should default on")
	}
	off := false
	p, err := Defaults().Apply(Patch{PartnerChallenges: &off})
	if err != nil || p.PartnerChallenges {
		t.Fatalf("Apply = %+v, %v; want it off", p, err)
	}
	p, err = p.Apply(Patch{})
	if err != nil || p.PartnerChallenges {
		t.Errorf("an empty patch changed it: %+v, %v", p, err)
	}
}
