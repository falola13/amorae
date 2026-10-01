package notifications

import "testing"

// Every kind sits on one side of the budget or the other, and somebody has
// decided which.
//
// AskedFor is load-bearing now: it decides whether the cap and quiet hours
// apply at all, and getting it wrong is how a live user stopped receiving
// event reminders. It is a list of kinds, and lists of kinds drift — this one
// went from nine to fifteen in a day. So kind sixteen fails here until its
// side is chosen on purpose rather than by default.
func TestEveryKindHasBeenDecidedAbout(t *testing.T) {
	// The person arranged it themselves, at a time they chose, so neither the
	// cap nor quiet hours applies.
	asked := map[string]bool{
		KindEventReminder:  true,
		KindPrayerReminder: true,
	}
	// Worthless once its moment has gone, so it is dropped rather than held.
	perishable := map[string]bool{
		KindChallenge: true,
	}

	for _, kind := range AllKinds {
		t.Run(kind, func(t *testing.T) {
			if got := AskedFor(kind); got != asked[kind] {
				t.Errorf("AskedFor = %v, want %v — decide which side this sits on", got, asked[kind])
			}
			if got := Perishable(kind); got != perishable[kind] {
				t.Errorf("Perishable = %v, want %v — decide whether it keeps", got, perishable[kind])
			}
			// Asked-for kinds never reach the perishable question, so saying
			// both would be a contradiction nobody would notice.
			if AskedFor(kind) && Perishable(kind) {
				t.Error("asked for and perishable: the second can never apply")
			}
		})
	}
}

// The same list, from the other end: a kind that exists but was never added
// to AllKinds is invisible to the test above.
func TestAllKindsIsComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, kind := range AllKinds {
		if seen[kind] {
			t.Errorf("%q is listed twice", kind)
		}
		seen[kind] = true
	}
	// Grows whenever a kind is added; the count is the reminder to add it to
	// AllKinds as well.
	if len(AllKinds) != 18 {
		t.Errorf("AllKinds has %d kinds; if that is right, update this and say why", len(AllKinds))
	}
}
