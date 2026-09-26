// Package notifications stores preferences and push subscriptions per
// person, not per couple (FR-NOTF-001); the worker does the sending.
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
	PrayerAnswered bool
	// When the second of the two of you finishes something.
	Together bool
	// A moment kept on this day in an earlier year.
	Memories bool
	// Halfway, and done — not every contribution, which is Goals.
	GoalMilestones bool
	// Whether "how was it?" follows a finished event — its own switch now,
	// but it used to ride on EventReminders, so a person who had turned
	// reminders off keeps the same silence after this one splits away.
	EventFollowups bool
	// Whether a partner's own event ("mine", not "together") is announced
	// to the other partner. Only ever about events somebody else made.
	PartnerEvents bool
	// What the create-event screen offers as already chosen. "1 hour
	// before" out of the box, matching what the screen always offered
	// before this was a preference; "" is a real choice — no default —
	// once somebody picks it, not what an unconfigured person gets.
	DefaultEventReminder string
	// Quiet hours in the person's own zone, as HH:MM. Both empty means off;
	// the window may wrap midnight.
	QuietFrom string
	QuietTo   string
	// Most notifications in one local day. 0 means no limit.
	DailyCap int
}

// Defaults are what someone gets before ever opening the screen. Goals and
// challenges default off since those are opt-in (FR-NOTF-006).
func Defaults() Preferences {
	return Preferences{
		NewWeek:              true,
		PrayerReminder:       true,
		ReminderTime:         "19:00",
		EventReminders:       true,
		ImportantDates:       true,
		Appreciation:         true,
		Journal:              true,
		Goals:                false,
		Challenges:           false,
		PrayerAnswered:       true,
		Together:             true,
		Memories:             true,
		GoalMilestones:       true,
		EventFollowups:       true,
		PartnerEvents:        true,
		DefaultEventReminder: "1 hour before",
		QuietFrom:            "22:00",
		QuietTo:              "07:00",
		DailyCap:             defaultDailyCap,
	}
}

// defaultDailyCap is what an unconfigured person gets: no limit. Six lived
// here until migration 00030_birthdays_and_uncapped.sql — nobody had ever
// actually chosen it, it just arrived unpicked alongside quiet hours
// (00024_notification_budget.sql), and 0 already means "no limit" to
// everything that reads DailyCap, so the unconfigured case now matches that.
const defaultDailyCap = 0

// maxDailyCap is the largest number the screen will accept. Higher than this
// is the same as no limit, so the screen says so instead.
const maxDailyCap = 20

// Patch changes a subset of settings; a nil field is left alone.
type Patch struct {
	NewWeek              *bool
	PrayerReminder       *bool
	ReminderTime         *string
	EventReminders       *bool
	ImportantDates       *bool
	Appreciation         *bool
	Journal              *bool
	Goals                *bool
	Challenges           *bool
	PrayerAnswered       *bool
	Together             *bool
	Memories             *bool
	GoalMilestones       *bool
	QuietFrom            *string
	QuietTo              *string
	DailyCap             *int
	EventFollowups       *bool
	PartnerEvents        *bool
	DefaultEventReminder *string
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
	if patch.DefaultEventReminder != nil {
		v := strings.TrimSpace(*patch.DefaultEventReminder)
		if !eventReminderOptions[v] {
			return Preferences{}, apperr.Validation(map[string]string{
				"default_event_reminder": "Pick one of the reminder options.",
			})
		}
		p.DefaultEventReminder = v
	}
	setBool(&p.NewWeek, patch.NewWeek)
	setBool(&p.PrayerReminder, patch.PrayerReminder)
	setBool(&p.EventReminders, patch.EventReminders)
	setBool(&p.ImportantDates, patch.ImportantDates)
	setBool(&p.Appreciation, patch.Appreciation)
	setBool(&p.Journal, patch.Journal)
	setBool(&p.Goals, patch.Goals)
	setBool(&p.PrayerAnswered, patch.PrayerAnswered)
	setBool(&p.Challenges, patch.Challenges)
	setBool(&p.Together, patch.Together)
	setBool(&p.Memories, patch.Memories)
	setBool(&p.GoalMilestones, patch.GoalMilestones)
	setBool(&p.EventFollowups, patch.EventFollowups)
	setBool(&p.PartnerEvents, patch.PartnerEvents)

	if patch.QuietFrom != nil || patch.QuietTo != nil {
		from, to := p.QuietFrom, p.QuietTo
		if patch.QuietFrom != nil {
			from = strings.TrimSpace(*patch.QuietFrom)
		}
		if patch.QuietTo != nil {
			to = strings.TrimSpace(*patch.QuietTo)
		}
		// One without the other is not a window, so it is refused rather
		// than guessed at.
		if (from == "") != (to == "") {
			return Preferences{}, apperr.Validation(map[string]string{
				"quiet_hours": "Set both a start and an end, or neither.",
			})
		}
		for field, value := range map[string]string{"quiet_from": from, "quiet_to": to} {
			if value != "" && !clockTime.MatchString(value) {
				return Preferences{}, apperr.Validation(map[string]string{
					field: "Use a time like 22:00.",
				})
			}
		}
		if from != "" && from == to {
			return Preferences{}, apperr.Validation(map[string]string{
				"quiet_hours": "Pick two different times.",
			})
		}
		p.QuietFrom, p.QuietTo = from, to
	}

	if patch.DailyCap != nil {
		if *patch.DailyCap < 0 || *patch.DailyCap > maxDailyCap {
			return Preferences{}, apperr.Validation(map[string]string{
				"daily_cap": fmt.Sprintf("Pick a number between 0 and %d.", maxDailyCap),
			})
		}
		p.DailyCap = *patch.DailyCap
	}
	return p, nil
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

// eventReminderOptions is the fixed set the create-event screen offers as a
// default; "" is "no default", not a phrase to match against.
var eventReminderOptions = map[string]bool{
	"":                  true,
	"at the time":       true,
	"10 minutes before": true,
	"30 minutes before": true,
	"1 hour before":     true,
	"2 hours before":    true,
	"the morning of":    true,
	"1 day before":      true,
}

// Subscription is one browser's push endpoint; the keys encrypt payloads
// only that browser can decrypt.
type Subscription struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Endpoint string
	P256dh   string
	Auth     string
}

// ValidateSubscription requires endpoint, p256dh, and auth together — the
// browser never sends only some of them.
func ValidateSubscription(endpoint, p256dh, auth string) (string, string, string, error) {
	endpoint, p256dh, auth = strings.TrimSpace(endpoint), strings.TrimSpace(p256dh), strings.TrimSpace(auth)

	fields := map[string]string{}
	if endpoint == "" {
		fields["endpoint"] = "Missing."
	} else if !strings.HasPrefix(endpoint, "https://") {
		// Must be https; anything else is malformed or malicious.
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

// ReminderPassed reports whether today's reminder has passed in the user's
// zone; the returned date keys notification_sends so a late or repeated
// tick still sends exactly once.
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

// reminderMorning is the hour used for day-anchored reminders ("morning of",
// "day before"); deliberately not the person's own evening prayer time.
const reminderMorning = 8

// eventLength is how long an event with a start but no end is taken to run,
// for deciding when it is over. Long enough that nobody is asked how dinner
// was while they are still eating it.
const eventLength = 2 * time.Hour

// EventOverAt is when an event has finished, in the couple's zone. An event
// with no times at all is a whole day, so it is over the next morning rather
// than at midnight — which is a time to be asleep, not to be asked anything.
func EventOverAt(date time.Time, startTime, endTime string, zone *time.Location) time.Time {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, zone)
	if end, ok := startOf(day, endTime, zone); ok {
		return end
	}
	if start, ok := startOf(day, startTime, zone); ok {
		return start.Add(eventLength)
	}
	return time.Date(day.Year(), day.Month(), day.Day()+1, reminderMorning, 0, 0, 0, zone)
}

// eventReminderLead matches reminder phrases like "an hour before".
var eventReminderLead = regexp.MustCompile(`^(\d{1,3}|a|an|the) (minute|hour|day)s? before$`)

// EventReminderAt is when an event's reminder fires, in the couple's zone
// (DEC-27, not each partner's own). With no start time, falls back to the
// morning of the day.
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
	// Day-counted reminders become morning-of, not a clock-time offset.
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
	KindNewWeek         = "new_week"
	KindWeekPublished   = "week_published"
	KindPrayerReminder  = "prayer_reminder"
	KindEventReminder   = "event_reminder"
	KindImportantDate   = "important_date"
	KindAppreciation    = "appreciation"
	KindJournal         = "journal"
	KindGoal            = "goal"
	KindChallenge       = "challenge"
	KindPrayerAnswered  = "prayer_answered"
	KindBothPrayed      = "both_prayed"
	KindBothMarked      = "both_marked"
	KindMemoryOnThisDay = "memory_on_this_day"
	KindEventOver       = "event_over"
	KindGoalCrossing    = "goal_crossing"
	// One partner's "together" event, told to the other. Never for "mine".
	KindEventAdded = "event_added"
)

// AllKinds is every kind there is. A kind missing from here is a kind no
// test can reason about, so adding one means adding it here too — which
// TestEveryKindHasBeenDecidedAbout then insists somebody has an opinion on.
var AllKinds = []string{
	KindNewWeek, KindWeekPublished, KindPrayerReminder, KindEventReminder,
	KindImportantDate, KindAppreciation, KindJournal, KindGoal, KindChallenge,
	KindPrayerAnswered, KindBothPrayed, KindBothMarked, KindMemoryOnThisDay,
	KindEventOver, KindGoalCrossing, KindEventAdded, KindNudge,
}

// OccursOn reports whether a date recurs on the given day (same month and
// day, any year); Feb 29 anniversaries fall back to Feb 28 in non-leap years.
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
