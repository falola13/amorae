package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// currentWeekStart matches prayers.StartOfWeek's Go computation, in the
// couple's zone (DEC-27), so a worker that was down on Sunday still finds
// the right week; the send is keyed on week id, so late is fine.
const currentWeekStart = `(
	($1 AT TIME ZONE c.timezone)::date
	- EXTRACT(DOW FROM ($1 AT TIME ZONE c.timezone))::int
)`

// Candidate is one person, with everything needed to decide and send,
// gathered in one query rather than one per person.
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

// CurrentWeekCandidates gathers everyone in a live couple with their current
// week and progress; ClaimSend, not this query, decides what's already sent.
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

// DueEventReminders casts a wide net (±2 days); EventReminderAt in Go
// decides the exact moment. start_time comes back as text since it's a wall
// clock in the couple's zone, not a driver TIME value.
func (r *PostgresRepository) DueEventReminders(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, e.id, e.title, c.timezone, e.date,
		       COALESCE(to_char(e.start_time, 'HH24:MI'), ''),
		       COALESCE(to_char(e.end_time, 'HH24:MI'), ''),
		       e.reminder,
		       COALESCE(p.event_reminders, true)
		FROM events e
		JOIN couples c ON c.id = e.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND NOT e.done
		  AND COALESCE(e.reminder, '') <> ''
		  AND e.date BETWEEN ($1 AT TIME ZONE c.timezone)::date - 2
		                 AND ($1 AT TIME ZONE c.timezone)::date + 2
	`, now)
	if err != nil {
		return nil, fmt.Errorf("finding events to remind about: %w", err)
	}
	defer rows.Close()

	var out []EventCandidate
	for rows.Next() {
		var c EventCandidate
		if err := rows.Scan(&c.UserID, &c.EventID, &c.Title, &c.Timezone, &c.Date,
			&c.StartTime, &c.EndTime, &c.Reminder, &c.Prefs.EventReminders); err != nil {
			return nil, fmt.Errorf("scanning event to remind about: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding events to remind about: %w", err)
	}
	return out, nil
}

// ImportantDates takes no window, unlike DueEventReminders — matching
// "today" against each couple's own zone and leap years isn't worth
// narrowing in SQL at current couple counts.
func (r *PostgresRepository) ImportantDates(ctx context.Context) ([]ImportantDateCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, ms.id, ms.title, c.timezone, ms.date, ms.reminder,
		       COALESCE(p.important_dates, true)
		FROM milestones ms
		JOIN couples c ON c.id = ms.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND ms.reminder
	`)
	if err != nil {
		return nil, fmt.Errorf("finding dates to remind about: %w", err)
	}
	defer rows.Close()

	var out []ImportantDateCandidate
	for rows.Next() {
		var c ImportantDateCandidate
		if err := rows.Scan(&c.UserID, &c.MilestoneID, &c.Title, &c.Timezone,
			&c.Date, &c.Reminder, &c.Prefs.ImportantDates); err != nil {
			return nil, fmt.Errorf("scanning date to remind about: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding dates to remind about: %w", err)
	}
	return out, nil
}

// RecentlyWritten unions appreciations and journal entries since `since`;
// the undo window travels as `settles`, decided in Go (ForWritten).
func (r *PostgresRepository) RecentlyWritten(ctx context.Context, since time.Time) ([]WrittenCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		WITH written AS (
			SELECT a.id, a.couple_id, a.from_id AS author_id, a.created_at, $2::text AS kind
			FROM appreciations a
			WHERE a.created_at >= $1
			UNION ALL
			SELECT j.id, j.couple_id, j.author_id, j.created_at, $3::text AS kind
			FROM journal_entries j
			WHERE j.created_at >= $1
			UNION ALL
			-- A goal somebody put something towards. Only while it is still
			-- going: nobody needs telling about a goal already finished.
			SELECT gp.id, g.couple_id, gp.user_id, gp.logged_at, $4::text AS kind
			FROM goal_progress gp
			JOIN goals g ON g.id = gp.goal_id AND NOT g.done
			WHERE gp.logged_at >= $1
			UNION ALL
			-- A prayer one of them marked answered. answered_at is set by an
			-- update rather than an insert, but it only ever goes from null
			-- to a time — editing the note deliberately leaves it alone — so
			-- it behaves like a creation here and cannot re-fire on an edit.
			SELECT pp.id, pw.couple_id, pp.answered_by, pp.answered_at, $5::text AS kind
			FROM prayer_points pp
			JOIN prayer_weeks pw ON pw.id = pp.week_id
			WHERE pp.answered_at >= $1 AND pp.answered_by IS NOT NULL
		)
		SELECT u.id, written.author_id, author.display_name, written.id, written.kind,
		       written.created_at,
		       COALESCE(p.appreciation, true), COALESCE(p.journal, true),
		       -- Goals default to off, unlike the rest: following one is
		       -- something you opt into (FR-NOTF-006).
		       COALESCE(p.goals, false),
		       COALESCE(p.prayer_answered, true),
		       COALESCE(g.title, '')
		FROM written
		LEFT JOIN goal_progress gpr ON gpr.id = written.id AND written.kind = $4
		LEFT JOIN goals g ON g.id = gpr.goal_id
		JOIN couples c ON c.id = written.couple_id AND c.dissolved_at IS NULL
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		JOIN users author ON author.id = written.author_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE u.id <> written.author_id
	`, since, KindAppreciation, KindJournal, KindGoal, KindPrayerAnswered)
	if err != nil {
		return nil, fmt.Errorf("finding what they have written: %w", err)
	}
	defer rows.Close()

	var out []WrittenCandidate
	for rows.Next() {
		var c WrittenCandidate
		if err := rows.Scan(&c.UserID, &c.AuthorID, &c.AuthorName, &c.ItemID, &c.Kind,
			&c.WrittenAt, &c.Prefs.Appreciation, &c.Prefs.Journal, &c.Prefs.Goals,
			&c.Prefs.PrayerAnswered, &c.Subject); err != nil {
			return nil, fmt.Errorf("scanning something written: %w", err)
		}
		if c.Kind == KindAppreciation {
			c.Settles = appreciationUndoWindow
		}
		if c.Kind == KindPrayerAnswered {
			c.Settles = answeredUndoWindow
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding what they have written: %w", err)
	}
	return out, nil
}

// LiveChallenges reads day/total/marked-today from SQL, since started_on
// and zone already sit together there; ForChallenge decides what to say.
func (r *PostgresRepository) LiveChallenges(ctx context.Context) ([]ChallengeCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, ch.id, ch.title, c.timezone,
		       (((now() AT TIME ZONE c.timezone)::date - ch.started_on) + 1)::int AS day,
		       (SELECT count(*) FROM challenge_days d WHERE d.challenge_id = ch.id)::int AS days,
		       EXISTS (
		           SELECT 1
		           FROM challenge_days d
		           JOIN challenge_progress pr ON pr.day_id = d.id AND pr.user_id = u.id
		           WHERE d.challenge_id = ch.id
		             AND d.n = ((now() AT TIME ZONE c.timezone)::date - ch.started_on) + 1
		       ) AS marked_today,
		       COALESCE(p.challenges, false)
		FROM challenges ch
		JOIN couples c ON c.id = ch.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("finding challenges to nudge about: %w", err)
	}
	defer rows.Close()

	var out []ChallengeCandidate
	for rows.Next() {
		var c ChallengeCandidate
		if err := rows.Scan(&c.UserID, &c.ChallengeID, &c.Title, &c.Timezone,
			&c.Day, &c.Days, &c.MarkedToday, &c.Prefs.Challenges); err != nil {
			return nil, fmt.Errorf("scanning a challenge to nudge about: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding challenges to nudge about: %w", err)
	}
	return out, nil
}

// appreciationUndoWindow mirrors appreciation.UndoWindow without importing
// it (BR-APPR-02), the same trade as statusDraft.
const appreciationUndoWindow = 30 * time.Second

// answeredUndoWindow is the pause before the other partner is told — long
// enough to catch a mis-tap, short enough not to sit on good news.
const answeredUndoWindow = time.Minute

// ClaimSend records the send as underway and reports whether this caller
// won the claim. Crash-safe: a send that then fails releases its claim, so
// the next tick retries instead of double-sending.
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
