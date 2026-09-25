package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func at(hhmm string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", "2026-09-25 "+hhmm)
	if err != nil {
		panic(err)
	}
	return t
}

func TestQuiet(t *testing.T) {
	night := Preferences{QuietFrom: "22:00", QuietTo: "07:00"}
	day := Preferences{QuietFrom: "09:00", QuietTo: "17:00"}

	tests := []struct {
		name  string
		prefs Preferences
		now   string
		want  bool
	}{
		{"before it starts", night, "21:59", false},
		{"the minute it starts", night, "22:00", true},
		// The window wraps midnight, which is the normal way to set one.
		{"after midnight", night, "03:00", true},
		{"the minute it ends", night, "07:00", false},
		{"a window inside one day", day, "12:00", true},
		{"outside a window inside one day", day, "08:00", false},
		{"unset means never quiet", Preferences{}, "03:00", false},
		{"half-set means never quiet", Preferences{QuietFrom: "22:00"}, "03:00", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Quiet(tc.prefs, time.UTC, at(tc.now)); got != tc.want {
				t.Errorf("Quiet = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQuiet_IsReadWhereThePersonIs(t *testing.T) {
	lagos, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		t.Skip("no timezone database")
	}
	prefs := Preferences{QuietFrom: "22:00", QuietTo: "07:00"}
	// 23:30 in Lagos, still 22:30 UTC.
	moment := at("22:30")
	if !Quiet(prefs, lagos, moment) {
		t.Error("it is night where they are")
	}
	if !Quiet(prefs, time.UTC, moment) {
		t.Error("it is night in UTC too")
	}
	// 21:30 Lagos is 20:30 UTC: quiet for neither.
	if Quiet(prefs, lagos, at("20:30")) {
		t.Error("their evening was treated as night")
	}
}

func TestOverCap(t *testing.T) {
	if OverCap(Preferences{DailyCap: 0}, 100) {
		t.Error("no cap should mean no cap")
	}
	if OverCap(Preferences{DailyCap: 3}, 2) {
		t.Error("under the cap was refused")
	}
	if !OverCap(Preferences{DailyCap: 3}, 3) {
		t.Error("the cap is a ceiling, not a target")
	}
}

func TestBudget_Allows(t *testing.T) {
	quiet := Budget{
		Prefs:    Preferences{QuietFrom: "22:00", QuietTo: "07:00"},
		Timezone: "UTC",
	}

	// A nudge about today held until morning is about a day that has gone, so
	// it is dropped rather than queued; something a partner did is still
	// worth reading at breakfast. Both are kinds the budget governs — the two
	// reminders are asked for and never reach here (TestBudget_AsksAreNotCapped).
	if send, keep, _ := quiet.Allows(KindChallenge, at("23:00")); send || keep {
		t.Errorf("perishable during quiet hours: send=%v keep=%v", send, keep)
	}
	if send, keep, _ := quiet.Allows(KindAppreciation, at("23:00")); send || !keep {
		t.Errorf("keepable during quiet hours: send=%v keep=%v", send, keep)
	}
	if send, _, _ := quiet.Allows(KindChallenge, at("12:00")); !send {
		t.Error("the middle of the day was treated as quiet hours")
	}

	full := Budget{Prefs: Preferences{DailyCap: 2}, Timezone: "UTC", SentToday: 2}
	if send, keep, _ := full.Allows(KindBothMarked, at("12:00")); send || !keep {
		t.Errorf("over cap: send=%v keep=%v", send, keep)
	}
}

func TestForBothMarked(t *testing.T) {
	c := BothMarkedCandidate{
		UserID: uuid.New(), ChallengeID: uuid.New(),
		Title: "Seven days of thanks", Day: 4, Days: 7,
		Prefs: Preferences{Together: true},
	}

	n, ok := ForBothMarked(c)
	if !ok {
		t.Fatal("a day they both finished said nothing")
	}
	if n.Kind != KindBothMarked {
		t.Errorf("kind = %q", n.Kind)
	}
	// Keyed by the day, not the person, so each of them is told once and a
	// later tick does not tell them again.
	if n.Key != c.ChallengeID.String()+":4" {
		t.Errorf("key = %q", n.Key)
	}

	last := c
	last.Day = 7
	done, _ := ForBothMarked(last)
	if done.Message.Body == n.Message.Body {
		t.Error("the last day reads the same as any other day")
	}

	off := c
	off.Prefs.Together = false
	if _, ok := ForBothMarked(off); ok {
		t.Error("it was sent to somebody who turned it off")
	}
}

func TestForBothPrayed(t *testing.T) {
	c := BothPrayedCandidate{
		UserID: uuid.New(), WeekID: uuid.New(), Points: 3,
		Prefs: Preferences{Together: true},
	}
	n, ok := ForBothPrayed(c)
	if !ok {
		t.Fatal("a finished week said nothing")
	}
	if n.Key != c.WeekID.String() {
		t.Errorf("key = %q, want the week", n.Key)
	}
	off := c
	off.Prefs.Together = false
	if _, ok := ForBothPrayed(off); ok {
		t.Error("it was sent to somebody who turned it off")
	}
}

// The cap has to hold within one tick as well as across them: several things
// can come due at once, and a budget read once at the start would let all of
// them through.
func TestTick_StopsAtTheCap(t *testing.T) {
	repo := newFakeRepo()
	repo.budget = Budget{Prefs: Preferences{DailyCap: 2}, Timezone: "UTC"}
	repo.subs = []Subscription{{Endpoint: "https://push.test/a", P256dh: "k", Auth: "a"}}

	me, challenge := uuid.New(), uuid.New()
	for day := 1; day <= 5; day++ {
		repo.bothMarked = append(repo.bothMarked, BothMarkedCandidate{
			UserID: me, ChallengeID: challenge, Title: "Seven days", Day: day, Days: 7,
			Prefs: Preferences{Together: true},
		})
	}

	sender := &fakeSender{}
	worker := NewWorker(repo, sender, func() time.Time { return at("12:00") }, quietLog())
	sent, err := worker.Tick(t.Context())
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if sent != 2 {
		t.Errorf("sent %d, want 2 — the cap did not hold inside one tick", sent)
	}
	if len(sender.sent) != 2 {
		t.Errorf("%d reached a device, want 2", len(sender.sent))
	}
}

// Held is not the same as sent: nothing is claimed, so the next tick still
// finds it due once the window has opened.
func TestTick_HoldingDoesNotConsumeTheClaim(t *testing.T) {
	repo := newFakeRepo()
	repo.budget = Budget{
		Prefs:    Preferences{QuietFrom: "22:00", QuietTo: "07:00"},
		Timezone: "UTC",
	}
	repo.subs = []Subscription{{Endpoint: "https://push.test/a", P256dh: "k", Auth: "a"}}
	repo.bothMarked = []BothMarkedCandidate{{
		UserID: uuid.New(), ChallengeID: uuid.New(), Title: "Seven days", Day: 1, Days: 7,
		Prefs: Preferences{Together: true},
	}}

	night := NewWorker(repo, &fakeSender{}, func() time.Time { return at("23:00") }, quietLog())
	if sent, err := night.Tick(t.Context()); err != nil || sent != 0 {
		t.Fatalf("sent %d during quiet hours (err %v)", sent, err)
	}
	if len(repo.claimed) != 0 {
		t.Fatal("a held notification was claimed, so morning will skip it")
	}

	sender := &fakeSender{}
	morning := NewWorker(repo, sender, func() time.Time { return at("08:00") }, quietLog())
	if sent, err := morning.Tick(t.Context()); err != nil || sent != 1 {
		t.Fatalf("sent %d after quiet hours (err %v)", sent, err)
	}
}

// The bug this fixes: a cap of 6 arrived on rows nobody had chosen it for, in
// the same deploy that took the kinds from nine to fifteen, and the first
// thing it ate was an event reminder — the one notification whose whole job is
// that you would otherwise miss something real.
func TestBudget_AsksAreNotCapped(t *testing.T) {
	full := Budget{
		Prefs:     Preferences{QuietFrom: "22:00", QuietTo: "07:00", DailyCap: 6},
		Timezone:  "UTC",
		SentToday: 6,
	}

	for _, kind := range []string{KindEventReminder, KindPrayerReminder} {
		t.Run("over the cap: "+kind, func(t *testing.T) {
			if send, _, why := full.Allows(kind, at("12:00")); !send {
				t.Errorf("held by %q — the person asked for this one", why)
			}
		})
		// An eleven o'clock reminder for an eleven o'clock plan is wanted at
		// eleven or not at all, so quiet hours do not hold it either.
		t.Run("inside quiet hours: "+kind, func(t *testing.T) {
			if send, _, why := full.Allows(kind, at("23:00")); !send {
				t.Errorf("held by %q — they chose this hour", why)
			}
		})
	}

	t.Run("everything unbidden is still governed", func(t *testing.T) {
		for _, kind := range []string{
			KindAppreciation, KindBothMarked, KindMemoryOnThisDay,
			KindGoalCrossing, KindEventOver, KindNudge,
		} {
			if send, _, _ := full.Allows(kind, at("12:00")); send {
				t.Errorf("%s ignored the cap", kind)
			}
		}
	})
}
