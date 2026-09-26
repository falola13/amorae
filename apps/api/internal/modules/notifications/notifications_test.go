package notifications

import (
	"testing"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

func TestApply_LeavesUntouchedFieldsAlone(t *testing.T) {
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

func TestApply_DefaultEventReminder(t *testing.T) {
	valid := []string{
		"", "at the time", "10 minutes before", "30 minutes before",
		"1 hour before", "2 hours before", "the morning of", "1 day before",
	}
	for _, in := range valid {
		if _, err := Defaults().Apply(Patch{DefaultEventReminder: &in}); err != nil {
			t.Errorf("Apply(%q) = %v, want nil", in, err)
		}
	}

	invalid := []string{"whenever", "5 minutes before", "1 week before"}
	for _, in := range invalid {
		_, err := Defaults().Apply(Patch{DefaultEventReminder: &in})
		if err == nil {
			t.Errorf("Apply(%q) was accepted", in)
			continue
		}
		appErr, ok := err.(*apperr.Error)
		if !ok || appErr.Fields["default_event_reminder"] == "" {
			t.Errorf("Apply(%q) should report the problem under default_event_reminder, got %v", in, err)
		}
	}

	t.Run("clearing it back to no default is allowed", func(t *testing.T) {
		set := "1 hour before"
		with, err := Defaults().Apply(Patch{DefaultEventReminder: &set})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		cleared := ""
		got, err := with.Apply(Patch{DefaultEventReminder: &cleared})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if got.DefaultEventReminder != "" {
			t.Errorf("DefaultEventReminder = %q, want cleared", got.DefaultEventReminder)
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
	if !d.EventFollowups || !d.PartnerEvents {
		t.Error("event_followups and partner_events should start on")
	}
	// Nobody ever chose six (migration 00030_birthdays_and_uncapped.sql) —
	// the unconfigured case now matches what 0 already means everywhere else.
	if d.DailyCap != 0 {
		t.Errorf("DailyCap = %d, want 0 (no limit)", d.DailyCap)
	}
	if d.DefaultEventReminder != "1 hour before" {
		t.Errorf("default_event_reminder = %q, want %q, matching what the screen always offered",
			d.DefaultEventReminder, "1 hour before")
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
		// 23:30 UTC is already the next day in Lagos; keying on the server's
		// date would double up or skip a reminder.
		date, _, err := ReminderPassed("19:00", lagos, time.Date(2026, 9, 23, 23, 30, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("ReminderPassed: %v", err)
		}
		if date != "2026-09-24" {
			t.Errorf("local date = %q, want 2026-09-24", date)
		}
	})

	t.Run("it is read in the person's own zone", func(t *testing.T) {
		// New York, not London (also UTC+1 in September, which would prove
		// nothing): 19:00 hasn't arrived there when it already has in Lagos.
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
