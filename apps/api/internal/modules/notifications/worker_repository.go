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

// DueEventReminders is every person in a live couple paired with an event of
// theirs whose reminder might be due.
//
// The window is wide on purpose. Which moment a phrase like "the day before"
// lands on is a rule, and rules live in Go (EventReminderAt) where they can
// be read and tested — so SQL narrows to the handful of events that could
// possibly matter and lets Go decide. Two days either side covers the longest
// lead the composer offers and leaves room for a worker that was down.
//
// start_time comes back as text rather than a TIME, because what it means is
// a wall clock in the couple's zone, and a driver's idea of a bare time is
// one conversion too many to reason about.
func (r *PostgresRepository) DueEventReminders(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, e.id, e.title, c.timezone, e.date,
		       COALESCE(to_char(e.start_time, 'HH24:MI'), ''),
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
			&c.StartTime, &c.Reminder, &c.Prefs.EventReminders); err != nil {
			return nil, fmt.Errorf("scanning event to remind about: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding events to remind about: %w", err)
	}
	return out, nil
}

// ImportantDates is every person in a live couple paired with a date that
// couple has asked to be reminded of.
//
// It takes no window, unlike DueEventReminders: whether a kept date comes
// round today is a question about a day of the year, and SQL narrowing it
// would mean comparing month and day across each couple's own timezone and
// each year's own February. A couple keeps a handful of these. When that
// stops being true, the narrowing to add is on (month, day) over the two
// local days any couple could currently be in.
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

// RecentlyWritten is every appreciation and journal entry written since
// `since`, paired with the partner who did not write it.
//
// One query over two tables because the question is one question: what has
// one of them written for the other lately. The undo window travels with each
// row as `settles`, so the rule about when a note is safe to announce lives
// in Go (ForWritten) and this only says which kind each row is.
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

// LiveChallenges is every person in a live couple paired with the challenge
// their couple is part way through.
//
// Which day of it today is comes back from SQL, because started_on and the
// couple's zone are already sitting together in these rows — and so does
// whether this person has marked that day, which is the only reason to stay
// quiet. Everything about when to say it stays in Go (ForChallenge).
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
// it: the worker reads that module's table, and one duration is a smaller
// thing to owe another module than a dependency is — the same trade as
// statusDraft above. BR-APPR-02 is where the thirty seconds is decided.
const appreciationUndoWindow = 30 * time.Second

// answeredUndoWindow is the pause before the other partner is told, and it
// exists because the screen offers Undo right beside the mark. A minute is
// long enough to catch the wrong prayer tapped, short enough that good news
// is not sat on.
const answeredUndoWindow = time.Minute

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
