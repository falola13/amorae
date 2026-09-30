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

// coupleLocalToday is "today" in the couple's own zone, matching
// prayers.StartOfDay — the day a point's weekdays bitmask is checked
// against, and the day a completion's prayed_on is compared to.
const coupleLocalToday = `(($1 AT TIME ZONE c.timezone)::date)`

// todaysWeekdayBit is the single bit of a prayer_points.weekdays mask that
// names today, mirroring prayers.ScheduledOn (bit n = weekday n, Sunday=0).
const todaysWeekdayBit = `(1 << EXTRACT(DOW FROM ` + coupleLocalToday + `)::int)`

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
	// Who published it, or the zero id for a draft or a week from before
	// either partner could (ForPublishedWeek falls back to SetterUserID).
	PublishedBy uuid.UUID
	WeekStatus  string
	// How many points the week has in total — used only to tell "nothing
	// set yet" (ForNewWeek) from "written" (ForPublishedWeek).
	Points int
	// Today's schedule and progress, couple-local — what ForReminder resets
	// on every day, rather than counting the whole week once and never again.
	TodayScheduled int
	TodayCompleted int
}

// CurrentWeekCandidates gathers everyone in a live couple with their current
// week and progress; ClaimSend, not this query, decides what's already sent.
func (r *PostgresRepository) CurrentWeekCandidates(ctx context.Context, now time.Time) ([]Candidate, error) {
	return r.candidates(ctx, `
		SELECT u.id, u.display_name, u.timezone,
		       COALESCE(p.new_week, true), COALESCE(p.prayer_reminder, true),
		       COALESCE(p.reminder_time, TIME '19:00'),
		       w.id, w.week_start, w.setter_user_id,
		       COALESCE(w.published_by, '00000000-0000-0000-0000-000000000000'::uuid),
		       w.status,
		       (SELECT count(*) FROM prayer_points pp WHERE pp.week_id = w.id),
		       (SELECT count(*) FROM prayer_points pp
		         WHERE pp.week_id = w.id AND (pp.weekdays::int & `+todaysWeekdayBit+`) <> 0),
		       (SELECT count(*) FROM prayer_points pp
		         JOIN prayer_completions pc ON pc.point_id = pp.id
		        WHERE pp.week_id = w.id AND pc.user_id = u.id
		          AND pc.prayed_on = `+coupleLocalToday+`
		          AND (pp.weekdays::int & `+todaysWeekdayBit+`) <> 0)
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
			&c.WeekID, &c.WeekStart, &c.SetterUserID, &c.PublishedBy, &c.WeekStatus,
			&c.Points, &c.TodayScheduled, &c.TodayCompleted); err != nil {
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

// DueEventReminders returns one candidate per reminder an event carries, so
// each is judged (and keyed) on its own. It casts a wide net (±2 days);
// EventReminderAt in Go decides the exact moment. start_time comes back as
// text since it's a wall clock in the couple's zone, not a driver TIME value.
func (r *PostgresRepository) DueEventReminders(ctx context.Context, now time.Time) ([]EventCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, e.id, e.title, c.timezone, e.date,
		       COALESCE(to_char(e.start_time, 'HH24:MI'), ''),
		       COALESCE(to_char(e.end_time, 'HH24:MI'), ''),
		       rem.reminder,
		       COALESCE(p.event_reminders, true)
		FROM events e
		CROSS JOIN LATERAL unnest(e.reminders) AS rem(reminder)
		JOIN couples c ON c.id = e.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND NOT e.done
		  AND NOT e.didnt_happen
		  AND rem.reminder <> ''
		  AND e.date BETWEEN ($1 AT TIME ZONE c.timezone)::date - 2
		                 AND ($1 AT TIME ZONE c.timezone)::date + 2
		  -- A "mine" event's reminder is its creator's alone.
		  AND (e.kind = 'together' OR e.created_by = u.id)
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
//
// Three sources, unioned so ForImportantDates never has to know which one a
// row came from: rows stored in milestones itself, the anniversary derived
// from couples.relationship_start_date, and each current member's birthday
// derived from their own profile — neither of the derived two is ever
// copied into milestones (see milestones.Milestone's Source field, and
// AnniversaryID/BirthdayID for the identical id computed on the Go side,
// matched by milestones.TestDerivedID_MatchesSQL). A birthday's row is
// joined to every current member and then filtered to exclude its own
// subject, so it reaches the other partner(s) and never the person whose
// day it is.
func (r *PostgresRepository) ImportantDates(ctx context.Context) ([]ImportantDateCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		WITH dates AS (
			SELECT ms.couple_id, ms.id, ms.title, ms.date, ms.reminder,
			       true AS year_known, NULL::uuid AS about
			FROM milestones ms
			WHERE ms.reminder
			UNION ALL
			SELECT c.id, md5(c.id::text || ':anniversary')::uuid, 'Our anniversary',
			       c.relationship_start_date, true, true, NULL::uuid
			FROM couples c
			WHERE c.relationship_start_date IS NOT NULL
			UNION ALL
			SELECT m.couple_id, md5(m.couple_id::text || ':birthday:' || u.id::text)::uuid,
			       u.display_name || '’s birthday',
			       make_date(COALESCE(u.birth_year, 2000)::int, u.birth_month, u.birth_day),
			       true, u.birth_year IS NOT NULL, u.id
			FROM couple_members m
			JOIN users u ON u.id = m.user_id
			WHERE m.ended_at IS NULL AND u.birth_month IS NOT NULL
		)
		SELECT u.id, d.id, d.title, c.timezone, d.date, d.reminder,
		       COALESCE(p.important_dates, true), d.year_known
		FROM dates d
		JOIN couples c ON c.id = d.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND (d.about IS NULL OR d.about <> u.id)
	`)
	if err != nil {
		return nil, fmt.Errorf("finding dates to remind about: %w", err)
	}
	defer rows.Close()

	var out []ImportantDateCandidate
	for rows.Next() {
		var c ImportantDateCandidate
		if err := rows.Scan(&c.UserID, &c.MilestoneID, &c.Title, &c.Timezone,
			&c.Date, &c.Reminder, &c.Prefs.ImportantDates, &c.YearKnown); err != nil {
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
			SELECT a.id, a.couple_id, a.from_id AS author_id, a.created_at, $2::text AS kind,
			       NULL::text AS subject
			FROM appreciations a
			WHERE a.created_at >= $1
			UNION ALL
			SELECT j.id, j.couple_id, j.author_id, j.created_at, $3::text AS kind,
			       NULL::text AS subject
			FROM journal_entries j
			WHERE j.created_at >= $1
			UNION ALL
			-- A goal somebody put something towards. Only while it is still
			-- going: nobody needs telling about a goal already finished.
			SELECT gp.id, g.couple_id, gp.user_id, gp.logged_at, $4::text AS kind,
			       NULL::text AS subject
			FROM goal_progress gp
			JOIN goals g ON g.id = gp.goal_id AND NOT g.done
			WHERE gp.logged_at >= $1
			UNION ALL
			-- A prayer one of them marked answered. answered_at is set by an
			-- update rather than an insert, but it only ever goes from null
			-- to a time — editing the note deliberately leaves it alone — so
			-- it behaves like a creation here and cannot re-fire on an edit.
			SELECT pp.id, pw.couple_id, pp.answered_by, pp.answered_at, $5::text AS kind,
			       NULL::text AS subject
			FROM prayer_points pp
			JOIN prayer_weeks pw ON pw.id = pp.week_id
			WHERE pp.answered_at >= $1 AND pp.answered_by IS NOT NULL
			UNION ALL
			-- A "together" event one partner just made for the two of them.
			-- "mine" is excluded here, not just gated on the way out: it is
			-- never this pipeline's news to carry. created_by IS NOT NULL
			-- keeps out anything from before ownership existed, which has
			-- nobody in particular to credit as having "added" it.
			SELECT e.id, e.couple_id, e.created_by, e.created_at, $6::text AS kind,
			       e.title AS subject
			FROM events e
			WHERE e.created_at >= $1 AND e.kind = 'together' AND e.created_by IS NOT NULL
		)
		SELECT u.id, written.author_id, author.display_name, written.id, written.kind,
		       written.created_at,
		       COALESCE(p.appreciation, true), COALESCE(p.journal, true),
		       -- Goals default to off, unlike the rest: following one is
		       -- something you opt into (FR-NOTF-006).
		       COALESCE(p.goals, false),
		       COALESCE(p.prayer_answered, true),
		       COALESCE(p.partner_events, true),
		       COALESCE(g.title, written.subject, '')
		FROM written
		LEFT JOIN goal_progress gpr ON gpr.id = written.id AND written.kind = $4
		LEFT JOIN goals g ON g.id = gpr.goal_id
		JOIN couples c ON c.id = written.couple_id AND c.dissolved_at IS NULL
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		JOIN users author ON author.id = written.author_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE u.id <> written.author_id
	`, since, KindAppreciation, KindJournal, KindGoal, KindPrayerAnswered, KindEventAdded)
	if err != nil {
		return nil, fmt.Errorf("finding what they have written: %w", err)
	}
	defer rows.Close()

	var out []WrittenCandidate
	for rows.Next() {
		var c WrittenCandidate
		if err := rows.Scan(&c.UserID, &c.AuthorID, &c.AuthorName, &c.ItemID, &c.Kind,
			&c.WrittenAt, &c.Prefs.Appreciation, &c.Prefs.Journal, &c.Prefs.Goals,
			&c.Prefs.PrayerAnswered, &c.Prefs.PartnerEvents, &c.Subject); err != nil {
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
// Only an active challenge is nudged about, and only for the day that has
// opened today (a mark, not a note on its own, is what settles it); once the
// calendar has run past the last day there is no longer a "today" to nudge.
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
		             AND pr.mark IS NOT NULL
		       ) AS marked_today,
		       COALESCE(p.challenges, false)
		FROM challenges ch
		JOIN couples c ON c.id = ch.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND ch.status = 'active'
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
