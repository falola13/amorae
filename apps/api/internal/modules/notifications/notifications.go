// Package notifications owns what each person wants to be told about, and
// which browsers have agreed to be told. It does not send anything: delivery
// belongs to the worker, which reads both of these.
//
// Everything here is per person. A couple shares a prayer week; it does not
// share a phone, a timezone for reminders, or an opinion about being buzzed
// at nine in the evening (FR-NOTF-001).
package notifications

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

var ErrNotFound = apperr.NotFound("subscription_not_found", "That subscription isn’t here.")

// Preferences is one person's settings. The zero value is not the default —
// Defaults() is, because "unset" and "everything off" are different things.
type Preferences struct {
	NewWeek        bool
	PrayerReminder bool
	// A wall-clock time, read in the user's own timezone.
	ReminderTime   string
	EventReminders bool
	ImportantDates bool
	Appreciation   bool
	Journal        bool
	Goals          bool
	Challenges     bool
}

// Defaults are what someone gets before they have ever opened the screen.
// Most are on, because a shared life that never tells you anything is not
// much use; goals and challenges are off, because following one is a thing
// you opt into rather than something that should start buzzing on its own
// (FR-NOTF-006).
func Defaults() Preferences {
	return Preferences{
		NewWeek:        true,
		PrayerReminder: true,
		ReminderTime:   "19:00",
		EventReminders: true,
		ImportantDates: true,
		Appreciation:   true,
		Journal:        true,
		Goals:          false,
		Challenges:     false,
	}
}

// Patch is a change to some of the settings. A nil field is left alone, so a
// client can send one switch without having to know the rest.
type Patch struct {
	NewWeek        *bool
	PrayerReminder *bool
	ReminderTime   *string
	EventReminders *bool
	ImportantDates *bool
	Appreciation   *bool
	Journal        *bool
	Goals          *bool
	Challenges     *bool
}

var clockTime = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// Apply returns these preferences with the patch laid over them, and refuses
// a reminder time that is not a 24-hour HH:MM (FR-NOTF-002.AC2).
func (p Preferences) Apply(patch Patch) (Preferences, error) {
	if patch.ReminderTime != nil {
		t := strings.TrimSpace(*patch.ReminderTime)
		if !clockTime.MatchString(t) {
			return Preferences{}, apperr.Validation(map[string]string{
				"reminder_time": "Use a time like 19:00.",
			})
		}
		p.ReminderTime = t
	}
	setBool(&p.NewWeek, patch.NewWeek)
	setBool(&p.PrayerReminder, patch.PrayerReminder)
	setBool(&p.EventReminders, patch.EventReminders)
	setBool(&p.ImportantDates, patch.ImportantDates)
	setBool(&p.Appreciation, patch.Appreciation)
	setBool(&p.Journal, patch.Journal)
	setBool(&p.Goals, patch.Goals)
	setBool(&p.Challenges, patch.Challenges)
	return p, nil
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

// Subscription is one browser that has agreed to receive push. The keys are
// the browser's own, and are only useful for encrypting a payload that only
// it can open.
type Subscription struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Endpoint string
	P256dh   string
	Auth     string
}

// ValidateSubscription checks the three parts a push send cannot work
// without. The browser supplies all of them together or not at all, so a
// missing one means a malformed client rather than a person's mistake — but
// it still gets a message a person could read.
func ValidateSubscription(endpoint, p256dh, auth string) (string, string, string, error) {
	endpoint, p256dh, auth = strings.TrimSpace(endpoint), strings.TrimSpace(p256dh), strings.TrimSpace(auth)

	fields := map[string]string{}
	if endpoint == "" {
		fields["endpoint"] = "Missing."
	} else if !strings.HasPrefix(endpoint, "https://") {
		// A push endpoint is always https; anything else is a client bug or
		// somebody pointing us at a server of their choosing.
		fields["endpoint"] = "Must be an https address."
	}
	if p256dh == "" {
		fields["keys.p256dh"] = "Missing."
	}
	if auth == "" {
		fields["keys.auth"] = "Missing."
	}
	if len(fields) > 0 {
		return "", "", "", apperr.Validation(fields)
	}
	return endpoint, p256dh, auth, nil
}

// ReminderPassed reports whether today's reminder time has come round yet in
// this person's own zone, and names the local date it belongs to.
//
// That date is the whole point. It is what the send record is keyed on
// (notification_sends.key), so "have they had today's reminder?" is a
// question the database answers rather than something the worker has to
// remember between ticks. A worker that restarts, runs late, or runs twice
// still sends exactly one.
//
// Compare this to asking "is it 19:00 right now?", which misses the reminder
// entirely whenever a tick runs a minute late.
func ReminderPassed(at string, zone *time.Location, now time.Time) (localDate string, passed bool, err error) {
	if !clockTime.MatchString(at) {
		return "", false, fmt.Errorf("reminder time %q is not HH:MM", at)
	}
	hour := int(at[0]-'0')*10 + int(at[1]-'0')
	minute := int(at[3]-'0')*10 + int(at[4]-'0')

	local := now.In(zone)
	due := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, zone)
	return local.Format(time.DateOnly), !local.Before(due), nil
}

// reminderMorning is the hour a reminder anchored to a day rather than to a
// time goes out — "the morning of", "the day before", and anything measured
// from the start of an event that has no start.
//
// Eight is early enough to be worth knowing and late enough not to wake
// anybody. It is deliberately not the person's own prayer reminder time,
// which is an evening by default and means a different thing.
const reminderMorning = 8

// eventReminderLead reads the phrases the event composer offers, plus the
// ones people wrote by hand before it offered anything ("an hour before").
var eventReminderLead = regexp.MustCompile(`^(\d{1,3}|a|an|the) (minute|hour|day)s? before$`)

// EventReminderAt is the moment an event's reminder is due, read in the zone
// the event's date and time are written in — the couple's (DEC-27), because
// an event happens at a place, not in whichever timezone each partner is
// standing in.
//
// It reports false when there is no reminder to send, or when the phrase is
// not one it can read. A phrase nobody can turn into a moment cannot be
// delivered, and guessing at one would be inventing a time to buzz somebody.
//
// An event with no start time is the interesting case: "an hour before" has
// nothing to be an hour before. Rather than drop the reminder somebody asked
// for, those fall back to the morning of the day — which is the only useful
// answer for something that takes the whole day anyway.
func EventReminderAt(date time.Time, startTime, reminder string, zone *time.Location) (time.Time, bool) {
	r := strings.ToLower(strings.TrimSpace(reminder))
	if r == "" {
		return time.Time{}, false
	}

	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, zone)
	morningOf := func(d time.Time) time.Time {
		return time.Date(d.Year(), d.Month(), d.Day(), reminderMorning, 0, 0, 0, zone)
	}
	start, timed := startOf(day, startTime, zone)

	if r == "the morning of" {
		return morningOf(day), true
	}
	if r == "at the time" {
		if !timed {
			return morningOf(day), true
		}
		return start, true
	}

	m := eventReminderLead.FindStringSubmatch(r)
	if m == nil {
		return time.Time{}, false
	}
	n := 1
	if m[1] != "a" && m[1] != "an" && m[1] != "the" {
		parsed, err := strconv.Atoi(m[1])
		if err != nil || parsed < 1 {
			return time.Time{}, false
		}
		n = parsed
	}
	// A reminder counted in days is a morning, not a time of day carried
	// backwards: "the day before" at 11pm is not what anybody means.
	if m[2] == "day" {
		return morningOf(day.AddDate(0, 0, -n)), true
	}
	if !timed {
		return morningOf(day), true
	}
	unit := time.Minute
	if m[2] == "hour" {
		unit = time.Hour
	}
	return start.Add(-time.Duration(n) * unit), true
}

// startOf turns an event's wall-clock start into an instant on its own day.
func startOf(day time.Time, hhmm string, zone *time.Location) (time.Time, bool) {
	if !clockTime.MatchString(hhmm) {
		return time.Time{}, false
	}
	hour := int(hhmm[0]-'0')*10 + int(hhmm[1]-'0')
	minute := int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, zone), true
}

// Kinds of notification, used as notification_sends.kind.
const (
	KindNewWeek        = "new_week"
	KindWeekPublished  = "week_published"
	KindPrayerReminder = "prayer_reminder"
	KindEventReminder  = "event_reminder"
	KindImportantDate  = "important_date"
)

// OccursOn reports whether a kept date comes round on the given day —
// same month, same day, any year.
//
// The twenty-ninth of February is the whole reason this is a function. Three
// years in four it does not exist, and a couple married on it should still
// hear from us: it moves to the twenty-eighth, which is the convention every
// calendar uses and the only one that does not skip an anniversary.
func OccursOn(date, day time.Time) bool {
	month, dayOfMonth := date.Month(), date.Day()
	if month == time.February && dayOfMonth == 29 && !isLeapYear(day.Year()) {
		dayOfMonth = 28
	}
	return day.Month() == month && day.Day() == dayOfMonth
}

func isLeapYear(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}
