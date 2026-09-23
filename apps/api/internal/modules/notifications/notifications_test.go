package notifications

import (
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestApply_LeavesUntouchedFieldsAlone(t *testing.T) {
	// The client sends one switch at a time, so a missing field has to mean
	// "leave it" — never "turn it off".
	start := Defaults()
	off := false

	got, err := start.Apply(Patch{Appreciation: &off})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got.Appreciation {
		t.Error("the field in the patch did not change")
	}
	if !got.NewWeek || !got.Journal || got.ReminderTime != start.ReminderTime {
		t.Error("a field nobody mentioned changed anyway")
	}
}

func TestApply_ReminderTime(t *testing.T) {
	valid := []string{"00:00", "09:05", "19:00", "23:59"}
	for _, in := range valid {
		if _, err := Defaults().Apply(Patch{ReminderTime: &in}); err != nil {
			t.Errorf("Apply(%q) = %v, want nil", in, err)
		}
	}

	invalid := []string{"", "7pm", "24:00", "19:60", "9:00", "19:00:00", "nineteen"}
	for _, in := range invalid {
		_, err := Defaults().Apply(Patch{ReminderTime: &in})
		if err == nil {
			t.Errorf("Apply(%q) was accepted", in)
			continue
		}
		appErr, ok := err.(*apperr.Error)
		if !ok || appErr.Fields["reminder_time"] == "" {
			t.Errorf("Apply(%q) should report the problem under reminder_time, got %v", in, err)
		}
	}

	t.Run("it is trimmed, not rejected, for stray spaces", func(t *testing.T) {
		padded := "  19:00  "
		got, err := Defaults().Apply(Patch{ReminderTime: &padded})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if got.ReminderTime != "19:00" {
			t.Errorf("ReminderTime = %q", got.ReminderTime)
		}
	})
}

func TestDefaults_AreMostlyOnButNotNoisy(t *testing.T) {
	d := Defaults()
	if !d.NewWeek || !d.Appreciation || !d.ImportantDates {
		t.Error("the things a couple would want to hear about are off by default")
	}
	// FR-NOTF-006: following a goal or a challenge is opted into, not
	// something that starts buzzing on its own.
	if d.Goals || d.Challenges {
		t.Error("goals and challenges should start off")
	}
}

func TestValidateSubscription(t *testing.T) {
	const endpoint = "https://fcm.googleapis.com/fcm/send/abc123"

	if _, _, _, err := ValidateSubscription(endpoint, "key", "secret"); err != nil {
		t.Errorf("a complete subscription was rejected: %v", err)
	}

	t.Run("every missing part is reported at once", func(t *testing.T) {
		_, _, _, err := ValidateSubscription("", "", "")
		appErr, ok := err.(*apperr.Error)
		if !ok {
			t.Fatalf("err = %T, want *apperr.Error", err)
		}
		for _, field := range []string{"endpoint", "keys.p256dh", "keys.auth"} {
			if appErr.Fields[field] == "" {
				t.Errorf("%s was not reported", field)
			}
		}
	})

	t.Run("an endpoint that is not https is refused", func(t *testing.T) {
		// Otherwise a client bug — or somebody choosing the address — points
		// our sends at a server of their own.
		if _, _, _, err := ValidateSubscription("http://example.com/push", "k", "a"); err == nil {
			t.Error("a plain http endpoint was accepted")
		}
	})
}

func TestReminderPassed(t *testing.T) {
	lagos, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		t.Fatalf("loading zone: %v", err)
	}
	// 19:00 in Lagos is 18:00 UTC.
	at := func(hour, minute int) time.Time {
		return time.Date(2026, 9, 23, hour, minute, 0, 0, time.UTC)
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before it", at(17, 0), false},
		{"the moment it arrives", at(18, 0), true},
		{"after it", at(20, 0), true},
		// The reason this is not an equality check: a tick that runs hours
		// late still owes the person their reminder.
		{"much later the same day", at(22, 30), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, passed, err := ReminderPassed("19:00", lagos, tc.now)
			if err != nil {
				t.Fatalf("ReminderPassed: %v", err)
			}
			if passed != tc.want {
				t.Errorf("passed = %v, want %v", passed, tc.want)
			}
		})
	}

	t.Run("the date is the person's own, not the server's", func(t *testing.T) {
		// 23:30 UTC is already the next day in Lagos. Keying the send on the
		// server's date would give someone two reminders on one of their
		// days and none on the next.
		date, _, err := ReminderPassed("19:00", lagos, time.Date(2026, 9, 23, 23, 30, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("ReminderPassed: %v", err)
		}
		if date != "2026-09-24" {
			t.Errorf("local date = %q, want 2026-09-24", date)
		}
	})

	t.Run("it is read in the person's own zone", func(t *testing.T) {
		// Not London: in September that is also UTC+1 and would prove
		// nothing. New York is four hours behind, where 19:00 has not come
		// round yet at the instant it already has in Lagos.
		newYork, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Fatalf("loading zone: %v", err)
		}
		now := at(18, 30)
		if _, passed, _ := ReminderPassed("19:00", lagos, now); !passed {
			t.Error("19:00 Lagos had not passed at 18:30 UTC")
		}
		if _, passed, _ := ReminderPassed("19:00", newYork, now); passed {
			t.Error("19:00 New York had passed while it was still early afternoon there")
		}
	})

	t.Run("a malformed time is an error, not a silent no", func(t *testing.T) {
		if _, _, err := ReminderPassed("7pm", lagos, at(23, 0)); err == nil {
			t.Error("ReminderPassed accepted a time it cannot read")
		}
	})
}
