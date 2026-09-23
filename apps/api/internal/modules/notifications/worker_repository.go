package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// currentWeekStart is the Sunday the couple's current prayer week begins on,
// read in the couple's own timezone (DEC-27) — the same date prayers.StartOfWeek
// arrives at in Go, so the two never disagree about which week it is.
//
// Matching the *current* week rather than "a week that starts today" matters:
// a worker that was down on Sunday morning would otherwise never tell anyone
// it was their week. The send is keyed on the week's id, so being late is
// fine and sending twice is impossible.
const currentWeekStart = `(
	($1 AT TIME ZONE c.timezone)::date
	- EXTRACT(DOW FROM ($1 AT TIME ZONE c.timezone))::int
)`

// Candidate is one person the worker might have something to tell, with
// everything needed to decide and to send — gathered in one query rather than
// one per person.
type Candidate struct {
	UserID uuid.UUID
	Name   string
	// The person's own zone, for their reminder time. Not the couple's.
	Timezone string
	Prefs    Preferences

	// The couple's current week, if there is one.
	WeekID       uuid.UUID
	WeekStart    time.Time
	SetterUserID uuid.UUID
	WeekStatus   string
	// How many points are in it, and how many this person has prayed.
	Points    int
	Completed int
}

// CurrentWeekCandidates is everyone in a live couple, with that couple's
// current prayer week and their own progress through it.
//
// One query for every kind of notification, rather than one per kind: which
// of them applies is a rule, and rules belong in Go where they can be read
// and tested. Nothing here asks "has this already been sent?" either —
// ClaimSend answers that, and it is the only answer that stays true when two
// workers ask at once.
func (r *PostgresRepository) CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error) {
	return r.candidates(ctx, `
		SELECT u.id, u.display_name, u.timezone,
		       COALESCE(p.new_week, true), COALESCE(p.prayer_reminder, true),
		       COALESCE(p.reminder_time, TIME '19:00'),
		       w.id, w.week_start, w.setter_user_id, w.status,
		       (SELECT count(*) FROM prayer_points pp WHERE pp.week_id = w.id),
		       (SELECT count(*) FROM prayer_completions pc
		         JOIN prayer_points pp ON pp.id = pc.point_id
		        WHERE pp.week_id = w.id AND pc.user_id = u.id)
		FROM prayer_weeks w
		JOIN couples c ON c.id = w.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND w.week_start = `+currentWeekStart+`
	`, now)
}

func (r *PostgresRepository) candidates(ctx context.Context, query string, args ...any) ([]Candidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding people to notify: %w", err)
	}
	defer rows.Close()

	var out []Candidate
	for rows.Next() {
		var c Candidate
		var reminder time.Time
		if err := rows.Scan(&c.UserID, &c.Name, &c.Timezone,
			&c.Prefs.NewWeek, &c.Prefs.PrayerReminder, &reminder,
			&c.WeekID, &c.WeekStart, &c.SetterUserID, &c.WeekStatus,
			&c.Points, &c.Completed); err != nil {
			return nil, fmt.Errorf("scanning person to notify: %w", err)
		}
		c.Prefs.ReminderTime = reminder.Format("15:04")
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding people to notify: %w", err)
	}
	return out, nil
}

// ClaimSend records that this notification is being sent, and reports whether
// this caller is the one that got to send it.
//
// Claiming before sending rather than recording after means a crash costs at
// most one notification instead of sending it twice on every restart. A send
// that then fails releases its claim, so the next tick tries again.
func (r *PostgresRepository) ClaimSend(ctx context.Context, userID uuid.UUID, kind, key string, at time.Time) (bool, error) {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_sends (user_id, kind, key, sent_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, kind, key) DO NOTHING
	`, userID, kind, key, at)
	if err != nil {
		return false, fmt.Errorf("claiming notification: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ReleaseSend undoes a claim whose send did not work, so it can be tried again.
func (r *PostgresRepository) ReleaseSend(ctx context.Context, userID uuid.UUID, kind, key string) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM notification_sends WHERE user_id = $1 AND kind = $2 AND key = $3
	`, userID, kind, key); err != nil {
		return fmt.Errorf("releasing notification claim: %w", err)
	}
	return nil
}
