package notifications

import "testing"

// The reason is the whole point of the change. "Holding a notification" without
// it cost a real debugging session: six kinds had been added and a cap of six
// arrived in the same deploy, nobody had chosen either, and the log said only
// that something was held.
func TestBudget_AllowsNamesWhichRuleHeldIt(t *testing.T) {
	prefs := Preferences{QuietFrom: "22:00", QuietTo: "07:00", DailyCap: 6}
	lagos := func(sent int) Budget {
		return Budget{Prefs: prefs, Timezone: "Africa/Lagos", SentToday: sent}
	}

	if _, _, why := lagos(6).Allows(KindEventReminder, at("12:00")); why != "daily_cap" {
		t.Errorf("why = %q, want daily_cap", why)
	}
	if _, _, why := lagos(0).Allows(KindEventReminder, at("23:00")); why != "quiet_hours" {
		t.Errorf("why = %q, want quiet_hours", why)
	}
	// Both true at once: the log must name the rule that actually decided,
	// not whichever the reader guesses.
	if _, _, why := lagos(6).Allows(KindEventReminder, at("23:00")); why != "quiet_hours" {
		t.Errorf("why = %q, want quiet_hours when both apply", why)
	}
	if send, _, why := lagos(0).Allows(KindEventReminder, at("12:00")); !send || why != "" {
		t.Errorf("send = %v, why = %q; a notification that was sent has nothing to explain", send, why)
	}
}
