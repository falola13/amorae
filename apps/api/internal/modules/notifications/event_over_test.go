package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func day(date string) time.Time {
	d, err := time.Parse(time.DateOnly, date)
	if err != nil {
		panic(err)
	}
	return d
}

func TestEventOverAt(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		want       string
	}{
		{"an end time is when it is over", "19:30", "22:00", "2026-09-25 22:00"},
		{"a start and no end runs two hours", "19:30", "", "2026-09-25 21:30"},
		// Midnight is a time to be asleep, not to be asked anything.
		{"a whole day is over the next morning", "", "", "2026-09-26 08:00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EventOverAt(day("2026-09-25"), tc.start, tc.end, time.UTC)
			if got.Format("2006-01-02 15:04") != tc.want {
				t.Errorf("over at %s, want %s", got.Format("2006-01-02 15:04"), tc.want)
			}
		})
	}
}

func endedEvent(start, end string) EventCandidate {
	return EventCandidate{
		UserID: uuid.New(), EventID: uuid.New(), Title: "Dinner at Terra",
		Timezone: "UTC", Date: day("2026-09-25"), StartTime: start, EndTime: end,
		Prefs: Preferences{EventFollowups: true},
	}
}

func TestForEventOver(t *testing.T) {
	c := endedEvent("19:30", "22:00")

	t.Run("after it ends, once", func(t *testing.T) {
		n, ok := ForEventOver(c, at("22:05"))
		if !ok {
			t.Fatal("nothing was asked")
		}
		if n.Message.Title != "How was it?" {
			t.Errorf("title = %q", n.Message.Title)
		}
		if n.Message.Body != "Dinner at Terra" {
			t.Errorf("body = %q", n.Message.Body)
		}
		// Keyed by the event, so it is asked once and never again.
		if n.Key != c.EventID.String() {
			t.Errorf("key = %q", n.Key)
		}
		// It leads to the event, where keeping it is already on offer.
		if n.Message.Path != "/together/events/"+c.EventID.String() {
			t.Errorf("path = %q", n.Message.Path)
		}
	})

	t.Run("not while it is still happening", func(t *testing.T) {
		if _, ok := ForEventOver(c, at("21:00")); ok {
			t.Error("it asked how dinner was during dinner")
		}
	})

	t.Run("not days later", func(t *testing.T) {
		late := at("22:05").Add(eventOverGrace)
		if _, ok := ForEventOver(c, late); ok {
			t.Error("it was still asking a day and a bit later")
		}
	})

	t.Run("not to somebody who turned event follow-ups off", func(t *testing.T) {
		off := c
		off.Prefs.EventFollowups = false
		if _, ok := ForEventOver(off, at("22:05")); ok {
			t.Error("it was sent anyway")
		}
	})

	// It rides on a switch of its own now, not on event_reminders — turning
	// reminders off should not silence "how was it?" too.
	t.Run("not tied to the reminder switch", func(t *testing.T) {
		reminderOnly := c
		reminderOnly.Prefs = Preferences{EventReminders: true, EventFollowups: false}
		if _, ok := ForEventOver(reminderOnly, at("22:05")); ok {
			t.Error("event_reminders alone was enough to send it")
		}
	})

	// Not wanting to be told beforehand says nothing about afterwards, so an
	// event with no reminder set still gets asked about.
	t.Run("a reminder was never the point", func(t *testing.T) {
		quiet := c
		quiet.Reminder = ""
		if _, ok := ForEventOver(quiet, at("22:05")); !ok {
			t.Error("an event with no reminder was skipped")
		}
	})

	t.Run("the question keeps, so it is not perishable", func(t *testing.T) {
		if Perishable(KindEventOver) {
			t.Error("asked at eleven it would be dropped rather than waiting for morning")
		}
	})
}
