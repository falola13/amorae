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

// BothPrayedCandidate is a week both partners have prayed all the way through.
type BothPrayedCandidate struct {
	UserID uuid.UUID
	WeekID uuid.UUID
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

// BothPrayedWeeks finds this week's prayers, where both have prayed all of
// them. Bounded by `since` for the same reason as BothMarkedDays.
func (r *PostgresRepository) BothPrayedWeeks(ctx context.Context, now, since time.Time) ([]BothPrayedCandidate, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT u.id, w.id,
		       (SELECT count(*) FROM prayer_points pp WHERE pp.week_id = w.id)::int,
		       COALESCE(p.together, true)
		FROM prayer_weeks w
		JOIN couples c ON c.id = w.couple_id
		JOIN couple_members m ON m.couple_id = c.id AND m.ended_at IS NULL
		JOIN users u ON u.id = m.user_id
		LEFT JOIN notification_preferences p ON p.user_id = u.id
		WHERE c.dissolved_at IS NULL
		  AND w.status = 'published'
		  AND w.week_start = `+currentWeekStart+`
		  AND (SELECT count(*) FROM prayer_points pp WHERE pp.week_id = w.id) > 0
		  AND (SELECT count(*) FROM couple_members m2
		        WHERE m2.couple_id = c.id AND m2.ended_at IS NULL) = 2
		  -- Every member has a completion for every point in the week.
		  AND NOT EXISTS (
		        SELECT 1
		        FROM couple_members m3
		        JOIN prayer_points pp ON pp.week_id = w.id
		        WHERE m3.couple_id = c.id AND m3.ended_at IS NULL
		          AND NOT EXISTS (
		                SELECT 1 FROM prayer_completions pc
		                 WHERE pc.point_id = pp.id AND pc.user_id = m3.user_id)
		      )
		  AND (SELECT max(pc.completed_at) FROM prayer_completions pc
		         JOIN prayer_points pp ON pp.id = pc.point_id
		        WHERE pp.week_id = w.id) > $2
	`, now, since)
	if err != nil {
		return nil, fmt.Errorf("finding weeks you both finished: %w", err)
	}
	defer rows.Close()

	var out []BothPrayedCandidate
	for rows.Next() {
		var c BothPrayedCandidate
		if err := rows.Scan(&c.UserID, &c.WeekID, &c.Points, &c.Prefs.Together); err != nil {
			return nil, fmt.Errorf("scanning a week you both finished: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("finding weeks you both finished: %w", err)
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
