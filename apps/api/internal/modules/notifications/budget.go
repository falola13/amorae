package notifications

import "time"

// Perishable reports whether a notification is worthless once its moment has
// passed. A daily nudge and an event reminder are; something a partner did is
// not. It decides what happens when a notification cannot be sent right now:
// a perishable one is skipped and forgotten, because the next tick will find
// it no longer due, and the rest wait for the window to open.
func Perishable(kind string) bool {
	switch kind {
	case KindPrayerReminder, KindEventReminder, KindChallenge:
		return true
	default:
		return false
	}
}

// Quiet reports whether now falls inside a person's quiet hours, read in
// their own zone. A window that wraps midnight is the normal case.
func Quiet(p Preferences, zone *time.Location, now time.Time) bool {
	if p.QuietFrom == "" || p.QuietTo == "" {
		return false
	}
	from, okFrom := minutesOfDay(p.QuietFrom)
	to, okTo := minutesOfDay(p.QuietTo)
	if !okFrom || !okTo || from == to {
		return false
	}
	local := now.In(zone)
	at := local.Hour()*60 + local.Minute()
	if from < to {
		return at >= from && at < to
	}
	return at >= from || at < to
}

// DayStart is the person's local midnight, the point a daily cap counts from.
func DayStart(zone *time.Location, now time.Time) time.Time {
	local := now.In(zone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
}

// OverCap reports whether this person has already had their day's worth.
func OverCap(p Preferences, sentToday int) bool {
	return p.DailyCap > 0 && sentToday >= p.DailyCap
}

func minutesOfDay(hhmm string) (int, bool) {
	if !clockTime.MatchString(hhmm) {
		return 0, false
	}
	hour := int(hhmm[0]-'0')*10 + int(hhmm[1]-'0')
	minute := int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')
	return hour*60 + minute, true
}
