package notifications

import (
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// seen is what one pass looked at, kept only to work out when the next one is
// worth running. It holds the pass's own slices, not copies.
type seen struct {
	people     []Candidate
	events     []EventCandidate
	ended      []EventCandidate
	dates      []ImportantDateCandidate
	challenges []ChallengeCandidate
	memories   []MemoryAnniversaryCandidate
	// When each notification held back this pass (quiet hours, the daily cap)
	// is let go.
	held []time.Time
	// A send failed and was released, so it is due again at the next tick.
	retry bool
}

// nextDueFrom is the earliest moment after now that a pass could have
// something to send, going by what this pass saw. Always at most
// safetyInterval away, so whatever it cannot foresee is late by no more than
// that. Writes are not foreseen here — Invalidate covers those.
func nextDueFrom(now time.Time, s seen) time.Time {
	next := now.Add(safetyInterval)
	consider := func(at time.Time) {
		if at.After(now) && at.Before(next) {
			next = at
		}
	}

	zones := map[string]*time.Location{}
	note := func(name string) {
		if _, known := zones[name]; !known {
			zones[name] = zoneOrUTC(name)
		}
	}

	for _, c := range s.people {
		note(c.Timezone)
		if !c.Prefs.PrayerReminder {
			continue
		}
		if minutes, ok := minutesOfDay(c.Prefs.ReminderTime); ok {
			consider(nextClock(now, zones[c.Timezone], minutes))
		}
	}
	for _, c := range s.events {
		note(c.Timezone)
		if !c.Prefs.EventReminders {
			continue
		}
		if at, ok := EventReminderAt(c.Date, c.StartTime, c.Reminder, zones[c.Timezone]); ok {
			consider(at)
		}
	}
	for _, c := range s.ended {
		note(c.Timezone)
		if c.Prefs.EventFollowups {
			consider(EventOverAt(c.Date, c.StartTime, c.EndTime, zones[c.Timezone]))
		}
	}
	for _, c := range s.dates {
		note(c.Timezone)
		// An anniversary is also announced at midnight (anniversaryMidnight).
		if c.Anniversary && c.Reminder && c.Prefs.ImportantDates {
			consider(nextClock(now, zones[c.Timezone], 0))
		}
	}
	for _, c := range s.challenges {
		note(c.Timezone)
	}
	for _, c := range s.memories {
		note(c.Timezone)
	}

	// Important dates, challenges, memories and the all-day events anchor on
	// the morning, in each couple's own zone.
	for _, zone := range zones {
		consider(nextClock(now, zone, reminderMorning*60))
	}
	for _, at := range s.held {
		consider(at)
	}
	return next
}

// nextClock is the first moment after now that the wall clock in zone reads
// the given minutes past midnight — today's if it is still ahead, else
// tomorrow's.
func nextClock(now time.Time, zone *time.Location, minutes int) time.Time {
	local := now.In(zone)
	at := time.Date(local.Year(), local.Month(), local.Day(), minutes/60, minutes%60, 0, 0, zone)
	if !at.After(now) {
		at = time.Date(local.Year(), local.Month(), local.Day()+1, minutes/60, minutes%60, 0, 0, zone)
	}
	return at
}

// heldUntil is when a notification held back by the budget can next go out,
// if that is known without asking the database: quiet hours end at quiet_to,
// and the daily cap turns over at the person's midnight. A perishable one is
// dropped, not kept, so nothing is waiting on it.
func heldUntil(why string, keep bool, b Budget, now time.Time) (time.Time, bool) {
	if !keep {
		return time.Time{}, false
	}
	zone := zoneOrUTC(b.Timezone)
	switch why {
	case "quiet_hours":
		if minutes, ok := minutesOfDay(b.Prefs.QuietTo); ok {
			return nextClock(now, zone, minutes), true
		}
	case "daily_cap":
		return nextClock(now, zone, 0), true
	}
	return time.Time{}, false
}

func zoneOrUTC(name string) *time.Location {
	zone, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return zone
}

// InvalidateOnWrite makes every successful change a signed-in person makes
// count as "something may now be due", so the next cron tick runs a pass
// (Worker.TickIfDue) instead of waiting for the clock. Reads and failed
// requests change nothing. Mount it inside authentication, on the routes that
// write; the tick endpoint is not one of them.
func InvalidateOnWrite(w *Worker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			sw := httpx.NewStatusWriter(rw)
			next.ServeHTTP(sw, r)

			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return
			}
			if sw.Status >= 200 && sw.Status < 300 {
				w.Invalidate()
			}
		})
	}
}
