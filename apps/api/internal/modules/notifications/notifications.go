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

// ReminderDue reports whether a personal reminder set for `at` should fire in
// the window ending now, read in the user's own zone.
//
// The window is how the worker stays honest about being late: it ticks
// hourly, so it asks "did this time pass since I last looked?" rather than
// "is it exactly 19:00 right now?", which would miss every reminder the
// moment a tick ran a second late.
func ReminderDue(at string, zone *time.Location, since, now time.Time) (bool, error) {
	if !clockTime.MatchString(at) {
		return false, fmt.Errorf("reminder time %q is not HH:MM", at)
	}
	hour, minute := int(at[0]-'0')*10+int(at[1]-'0'), int(at[3]-'0')*10+int(at[4]-'0')

	local := now.In(zone)
	due := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, zone)
	if due.After(local) {
		// Today's has not come round yet; yesterday's may still be unsent.
		due = due.AddDate(0, 0, -1)
	}
	return due.After(since) && !due.After(now), nil
}
