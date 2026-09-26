package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BothMarkedCandidate is a challenge day both partners have marked done.
type BothMarkedCandidate struct {
	UserID      uuid.UUID
	ChallengeID uuid.UUID
	Title       string
	Day         int
	Days        int
	Prefs       Preferences
}

// BothPrayedCandidate is one day, in the current week, that both partners
// have prayed all the way through.
type BothPrayedCandidate struct {
	UserID uuid.UUID
	WeekID uuid.UUID
	// The couple-local date this is about — "today", since that's the only
	// day BothPrayedWeeks ever checks.
	Date   time.Time
	Points int
	Prefs  Preferences
}

// BothMarkedDays finds days where the second partner has just finished.
//
// `since` bounds it to marks made recently: without that, shipping this
// would announce every day either of them ever completed, and a couple who
// finished a challenge last month would get a week of congratulations.
func (r *PostgresRepository) BothMarkedDays(ctx context.Context, since time.Time) ([]BothMarkedCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, ch.id, ch.title, d.n,
		       (SELECT count(*) FROM challenge_days x WHERE x.challenge_id = ch.id)::int,
		       COALESCE(p.together, true)
		FROM challenge_days d
		JOIN challenges ch ON ch.id = d.challenge_id
		JOIN couples c ON c.id = ch.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND (SELECT count(*) FROM couple_members m2
		        WHERE m2.couple_id = c.id AND m2.ended_at IS NULL) = 2
		  AND (SELECT count(*) FROM challenge_progress pr
		         JOIN couple_members m3 ON m3.user_id = pr.user_id
		                               AND m3.couple_id = c.id AND m3.ended_at IS NULL
		        WHERE pr.day_id = d.id AND pr.mark = 'done') = 2
		  AND (SELECT max(pr.marked_at) FROM challenge_progress pr
		         JOIN couple_members m4 ON m4.user_id = pr.user_id
		                               AND m4.couple_id = c.id AND m4.ended_at IS NULL
		        WHERE pr.day_id = d.id AND pr.mark = 'done') > $1
	`, since)
	if err != nil {
		return nil, fmt.Errorf("finding days you both finished: %w", err)
	}
	defer rows.Close()

	var out []BothMarkedCandidate
	for rows.Next() {
		var c BothMarkedCandidate
		if err := rows.Scan(&c.UserID, &c.ChallengeID, &c.Title, &c.Day, &c.Days,
			&c.Prefs.Together); err != nil {
			return nil, fmt.Errorf("scanning a day you both finished: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding days you both finished: %w", err)
	}
	return out, nil
}

// BothPrayedWeeks finds today's schedule in the current week, where both
// partners have prayed everything scheduled for today. Bounded by `since`
// for the same reason as BothMarkedDays — otherwise a couple who finished
// yesterday would still be "both finished" a week later.
func (r *PostgresRepository) BothPrayedWeeks(ctx context.Context, now, since time.Time) ([]BothPrayedCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, w.id, `+coupleLocalToday+`,
		       (SELECT count(*) FROM prayer_points pp
		         WHERE pp.week_id = w.id AND (pp.weekdays::int & `+todaysWeekdayBit+`) <> 0)::int,
		       COALESCE(p.together, true)
		FROM prayer_weeks w
		JOIN couples c ON c.id = w.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND w.status = 'published'
		  AND w.week_start = `+currentWeekStart+`
		  AND (SELECT count(*) FROM prayer_points pp
		        WHERE pp.week_id = w.id AND (pp.weekdays::int & `+todaysWeekdayBit+`) <> 0) > 0
		  AND (SELECT count(*) FROM couple_members m2
		        WHERE m2.couple_id = c.id AND m2.ended_at IS NULL) = 2
		  -- Every member has, for today, a completion for every point
		  -- scheduled today.
		  AND NOT EXISTS (
		        SELECT 1
		        FROM couple_members m3
		        JOIN prayer_points pp ON pp.week_id = w.id
		                              AND (pp.weekdays::int & `+todaysWeekdayBit+`) <> 0
		        WHERE m3.couple_id = c.id AND m3.ended_at IS NULL
		          AND NOT EXISTS (
		                SELECT 1 FROM prayer_completions pc
		                 WHERE pc.point_id = pp.id AND pc.user_id = m3.user_id
		                   AND pc.prayed_on = `+coupleLocalToday+`)
		      )
		  AND (SELECT max(pc.completed_at) FROM prayer_completions pc
		         JOIN prayer_points pp ON pp.id = pc.point_id
		        WHERE pp.week_id = w.id AND pc.prayed_on = `+coupleLocalToday+`) > $2
	`, now, since)
	if err != nil {
		return nil, fmt.Errorf("finding days you both finished: %w", err)
	}
	defer rows.Close()

	var out []BothPrayedCandidate
	for rows.Next() {
		var c BothPrayedCandidate
		if err := rows.Scan(&c.UserID, &c.WeekID, &c.Date, &c.Points, &c.Prefs.Together); err != nil {
			return nil, fmt.Errorf("scanning a day you both finished: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding days you both finished: %w", err)
	}
	return out, nil
}

// BudgetFor is everything the gate needs about one person: their quiet
// hours, their cap, and how much of today they have already had. Counted
// from midnight where they are, not where the server is.
func (r *PostgresRepository) BudgetFor(ctx context.Context, userID uuid.UUID, now time.Time) (Budget, error) {
	var b Budget
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT COALESCE(to_char(p.quiet_from, 'HH24:MI'), ''),
		       COALESCE(to_char(p.quiet_to, 'HH24:MI'), ''),
		       COALESCE(p.daily_cap, $3),
		       u.timezone,
		       (SELECT count(*) FROM notification_sends s
		         WHERE s.user_id = u.id
		           AND s.sent_at >= date_trunc('day', ($2 AT TIME ZONE u.timezone))
		                            AT TIME ZONE u.timezone)
		FROM users u
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE u.id = $1
	`, userID, now, defaultDailyCap).Scan(
		&b.Prefs.QuietFrom, &b.Prefs.QuietTo, &b.Prefs.DailyCap, &b.Timezone, &b.SentToday)
	if err != nil {
		return Budget{}, fmt.Errorf("reading a notification budget: %w", err)
	}
	return b, nil
}

// NudgeTarget answers who the other partner is, and what this one is called
// — the only two facts a nudge carries. A person with no live partner gets
// the zero id rather than an error: not being paired yet is a state, not a
// fault.
func (r *PostgresRepository) NudgeTarget(ctx context.Context, senderID uuid.UUID) (uuid.UUID, string, error) {
	var partnerID uuid.UUID
	var name string
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT COALESCE(other.user_id, '00000000-0000-0000-0000-000000000000'::uuid), u.display_name
		FROM users u
		LEFT JOIN couple_members mine ON mine.user_id = u.id AND mine.ended_at IS NULL
		LEFT JOIN couples c ON c.id = mine.couple_id AND c.dissolved_at IS NULL
		LEFT JOIN couple_members other ON other.couple_id = c.id
		                              AND other.ended_at IS NULL
		                              AND other.user_id <> u.id
		WHERE u.id = $1
	`, senderID).Scan(&partnerID, &name)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("finding who to nudge: %w", err)
	}
	return partnerID, name, nil
}

// CountSends counts one kind of notification already sent to somebody since
// a moment — the day's allowance, for anything that has one.
func (r *PostgresRepository) CountSends(ctx context.Context, userID uuid.UUID, kind string, since time.Time) (int, error) {
	var n int
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT count(*) FROM notification_sends
		 WHERE user_id = $1 AND kind = $2 AND sent_at >= $3
	`, userID, kind, since).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting notifications: %w", err)
	}
	return n, nil
}

// MemoryAnniversaryCandidate is one kept moment, for one of the two people
// it belongs to.
type MemoryAnniversaryCandidate struct {
	UserID   uuid.UUID
	MemoryID uuid.UUID
	Title    string
	Timezone string
	Date     time.Time
	Prefs    Preferences
}

// MemoriesOnThisDay narrows to the right month and to years already past;
// which day it is, and what to do about the 29th of February, is decided in
// Go by OccursOn so there is one rule rather than two.
func (r *PostgresRepository) MemoriesOnThisDay(ctx context.Context) ([]MemoryAnniversaryCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, mem.id, mem.title, c.timezone, mem.date,
		       COALESCE(p.memories, true)
		FROM memories mem
		JOIN couples c ON c.id = mem.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND EXTRACT(MONTH FROM mem.date) =
		      EXTRACT(MONTH FROM (now() AT TIME ZONE c.timezone))
		  AND mem.date < (now() AT TIME ZONE c.timezone)::date
	`)
	if err != nil {
		return nil, fmt.Errorf("finding moments from this day: %w", err)
	}
	defer rows.Close()

	var out []MemoryAnniversaryCandidate
	for rows.Next() {
		var c MemoryAnniversaryCandidate
		if err := rows.Scan(&c.UserID, &c.MemoryID, &c.Title, &c.Timezone,
			&c.Date, &c.Prefs.Memories); err != nil {
			return nil, fmt.Errorf("scanning a moment from this day: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding moments from this day: %w", err)
	}
	return out, nil
}

// EndedEvents finds events that have just been and gone, so the two of them
// can be asked whether it is worth keeping. Unlike the reminder query this
// does not care whether a reminder was set: not wanting to be told before
// says nothing about after.
func (r *PostgresRepository) EndedEvents(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, e.id, e.title, c.timezone, e.date,
		       COALESCE(to_char(e.start_time, 'HH24:MI'), ''),
		       COALESCE(to_char(e.end_time, 'HH24:MI'), ''),
		       COALESCE(e.reminder, ''),
		       COALESCE(p.event_reminders, true)
		FROM events e
		JOIN couples c ON c.id = e.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND NOT e.done
		  AND e.date BETWEEN ($1 AT TIME ZONE c.timezone)::date - 2
		                 AND ($1 AT TIME ZONE c.timezone)::date
		  -- A "mine" event is only ever its creator's to be asked about.
		  AND (e.kind = 'together' OR e.created_by = u.id)
	`, now)
	if err != nil {
		return nil, fmt.Errorf("finding events that are over: %w", err)
	}
	defer rows.Close()

	var out []EventCandidate
	for rows.Next() {
		var c EventCandidate
		if err := rows.Scan(&c.UserID, &c.EventID, &c.Title, &c.Timezone, &c.Date,
			&c.StartTime, &c.EndTime, &c.Reminder, &c.Prefs.EventReminders); err != nil {
			return nil, fmt.Errorf("scanning an event that is over: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding events that are over: %w", err)
	}
	return out, nil
}

// GoalCrossingCandidate is a goal and how far along it is, for one of the
// two people working at it.
type GoalCrossingCandidate struct {
	UserID uuid.UUID
	GoalID uuid.UUID
	Title  string
	Target int64
	Total  int64
	Prefs  Preferences
}

// GoalsJustMovedOn finds goals somebody has put something towards recently.
// Whether that crossed anything is decided in Go, so the thresholds live in
// one place rather than in SQL.
//
// `since` keeps it to recent contributions: without it, shipping this would
// announce halfway for every goal already past it.
func (r *PostgresRepository) GoalsJustMovedOn(ctx context.Context, since time.Time) ([]GoalCrossingCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, g.id, g.title, g.target,
		       COALESCE((SELECT sum(gp.amount) FROM goal_progress gp
		                  WHERE gp.goal_id = g.id), 0)::bigint,
		       COALESCE(p.goal_milestones, true)
		FROM goals g
		JOIN couples c ON c.id = g.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND g.target > 0
		  AND EXISTS (SELECT 1 FROM goal_progress gp
		               WHERE gp.goal_id = g.id AND gp.logged_at > $1)
	`, since)
	if err != nil {
		return nil, fmt.Errorf("finding goals that have moved on: %w", err)
	}
	defer rows.Close()

	var out []GoalCrossingCandidate
	for rows.Next() {
		var c GoalCrossingCandidate
		if err := rows.Scan(&c.UserID, &c.GoalID, &c.Title, &c.Target, &c.Total,
			&c.Prefs.GoalMilestones); err != nil {
			return nil, fmt.Errorf("scanning a goal that has moved on: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding goals that have moved on: %w", err)
	}
	return out, nil
}
