package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// PreferencesFor returns this person's saved preferences. No row is not an
// error; the service supplies the defaults.
func (r *PostgresRepository) PreferencesFor(ctx context.Context, userID uuid.UUID) (Preferences, bool, error) {
	var p Preferences
	var reminder time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT new_week, prayer_reminder, reminder_time, event_reminders,
		       important_dates, appreciation, journal, goals, challenges,
		       prayer_answered, together, memories, goal_milestones,
		       event_followups, partner_events, default_event_reminder, nudges,
		       COALESCE(to_char(quiet_from, 'HH24:MI'), ''),
		       COALESCE(to_char(quiet_to, 'HH24:MI'), ''),
		       daily_cap
		FROM notification_preferences WHERE user_id = $1
	`, userID).Scan(&p.NewWeek, &p.PrayerReminder, &reminder, &p.EventReminders,
		&p.ImportantDates, &p.Appreciation, &p.Journal, &p.Goals, &p.Challenges,
		&p.PrayerAnswered, &p.Together, &p.Memories, &p.GoalMilestones,
		&p.EventFollowups, &p.PartnerEvents, &p.DefaultEventReminder, &p.Nudges,
		&p.QuietFrom, &p.QuietTo, &p.DailyCap)
	if errors.Is(err, pgx.ErrNoRows) {
		return Defaults(), false, nil
	}
	if err != nil {
		return Preferences{}, false, fmt.Errorf("loading notification preferences: %w", err)
	}
	p.ReminderTime = reminder.Format("15:04")
	return p, true, nil
}

// SavePreferences writes the whole row; the service has already merged the
// patch, so no partial update happens here.
func (r *PostgresRepository) SavePreferences(ctx context.Context, userID uuid.UUID, p Preferences, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_preferences
			(user_id, new_week, prayer_reminder, reminder_time, event_reminders,
			 important_dates, appreciation, journal, goals, challenges,
			 prayer_answered, together, memories, goal_milestones,
			 event_followups, partner_events, default_event_reminder, nudges,
			 quiet_from, quiet_to, daily_cap, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		        $15, $16, $17, $18,
		        NULLIF($19, '')::time, NULLIF($20, '')::time, $21, $22, $22)
		ON CONFLICT (user_id) DO UPDATE SET
			new_week = EXCLUDED.new_week,
			prayer_reminder = EXCLUDED.prayer_reminder,
			reminder_time = EXCLUDED.reminder_time,
			event_reminders = EXCLUDED.event_reminders,
			important_dates = EXCLUDED.important_dates,
			appreciation = EXCLUDED.appreciation,
			journal = EXCLUDED.journal,
			goals = EXCLUDED.goals,
			challenges = EXCLUDED.challenges,
			prayer_answered = EXCLUDED.prayer_answered,
			together = EXCLUDED.together,
			memories = EXCLUDED.memories,
			goal_milestones = EXCLUDED.goal_milestones,
			event_followups = EXCLUDED.event_followups,
			partner_events = EXCLUDED.partner_events,
			default_event_reminder = EXCLUDED.default_event_reminder,
			nudges = EXCLUDED.nudges,
			quiet_from = EXCLUDED.quiet_from,
			quiet_to = EXCLUDED.quiet_to,
			daily_cap = EXCLUDED.daily_cap,
			updated_at = EXCLUDED.updated_at
	`, userID, p.NewWeek, p.PrayerReminder, p.ReminderTime, p.EventReminders,
		p.ImportantDates, p.Appreciation, p.Journal, p.Goals, p.Challenges,
		p.PrayerAnswered, p.Together, p.Memories, p.GoalMilestones,
		p.EventFollowups, p.PartnerEvents, p.DefaultEventReminder, p.Nudges,
		p.QuietFrom, p.QuietTo, p.DailyCap, at); err != nil {
		return fmt.Errorf("saving notification preferences: %w", err)
	}
	return nil
}

// Subscribe upserts by endpoint, reassigning it to this user if it belonged
// to someone else (shared or handed-on devices).
func (r *PostgresRepository) Subscribe(ctx context.Context, sub Subscription, at time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generating subscription id: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth
	`, id, sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, at); err != nil {
		return fmt.Errorf("saving push subscription: %w", err)
	}
	return nil
}

// SubscriptionsFor is every browser this person has agreed to be reached on.
func (r *PostgresRepository) SubscriptionsFor(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, user_id, endpoint, p256dh, auth
		FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing push subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, fmt.Errorf("scanning push subscription: %w", err)
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing push subscriptions: %w", err)
	}
	return subs, nil
}

// Unsubscribe forgets a browser; called by the worker when a push service
// reports it gone (FR-NOTF-004).
func (r *PostgresRepository) Unsubscribe(ctx context.Context, endpoint string) error {
	if _, err := r.db.Q(ctx).Exec(ctx,
		`DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint); err != nil {
		return fmt.Errorf("removing push subscription: %w", err)
	}
	return nil
}

// MarkSent records that a send to this browser worked, which is the only
// evidence we have that a subscription is still alive.
func (r *PostgresRepository) MarkSent(ctx context.Context, endpoint string, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx,
		`UPDATE push_subscriptions SET last_sent_at = $2 WHERE endpoint = $1`, endpoint, at); err != nil {
		return fmt.Errorf("recording push send: %w", err)
	}
	return nil
}

// RecordInbox appends one row to this person's own history, then trims it
// back to inboxRetention — two statements rather than one, since the row
// this insert just added is exactly what the trim below must be able to see.
func (r *PostgresRepository) RecordInbox(ctx context.Context, userID uuid.UUID, kind, title, body, path string, at time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generating an inbox id: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_inbox (id, user_id, kind, title, body, path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, userID, kind, title, body, path, at); err != nil {
		return fmt.Errorf("recording a notification in the inbox: %w", err)
	}
	if _, err := r.db.Q(ctx).Exec(ctx, `
		DELETE FROM notification_inbox
		 WHERE user_id = $1
		   AND id NOT IN (
		         SELECT id FROM notification_inbox
		          WHERE user_id = $1
		          ORDER BY created_at DESC
		          LIMIT $2
		       )
	`, userID, inboxRetention); err != nil {
		return fmt.Errorf("trimming the inbox: %w", err)
	}
	return nil
}

// Inbox is this person's own history, newest first — at most inboxRetention
// rows, since RecordInbox never lets there be more.
func (r *PostgresRepository) Inbox(ctx context.Context, userID uuid.UUID) ([]InboxItem, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, kind, title, body, path, created_at, read_at IS NOT NULL
		FROM notification_inbox
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing an inbox: %w", err)
	}
	defer rows.Close()

	var out []InboxItem
	for rows.Next() {
		var it InboxItem
		if err := rows.Scan(&it.ID, &it.Kind, &it.Title, &it.Body, &it.Path,
			&it.CreatedAt, &it.Read); err != nil {
			return nil, fmt.Errorf("scanning an inbox row: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing an inbox: %w", err)
	}
	return out, nil
}

// MarkInboxRead marks every one of this person's rows read at once, as of
// now — a retried call simply repeats a no-op on rows already read.
func (r *PostgresRepository) MarkInboxRead(ctx context.Context, userID uuid.UUID, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE notification_inbox SET read_at = $2
		 WHERE user_id = $1 AND read_at IS NULL
	`, userID, at); err != nil {
		return fmt.Errorf("marking an inbox read: %w", err)
	}
	return nil
}
