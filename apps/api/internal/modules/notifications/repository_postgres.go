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

// PreferencesFor is what this person has chosen, and whether they have ever
// chosen anything. Someone who has never opened the screen has no row, which
// is not an error — the service answers with the defaults.
func (r *PostgresRepository) PreferencesFor(ctx context.Context, userID uuid.UUID) (Preferences, bool, error) {
	var p Preferences
	var reminder time.Time
	err := r.db.Q(ctx).QueryRow(ctx, `
		SELECT new_week, prayer_reminder, reminder_time, event_reminders,
		       important_dates, appreciation, journal, goals, challenges
		FROM notification_preferences WHERE user_id = $1
	`, userID).Scan(&p.NewWeek, &p.PrayerReminder, &reminder, &p.EventReminders,
		&p.ImportantDates, &p.Appreciation, &p.Journal, &p.Goals, &p.Challenges)
	if errors.Is(err, pgx.ErrNoRows) {
		return Defaults(), false, nil
	}
	if err != nil {
		return Preferences{}, false, fmt.Errorf("loading notification preferences: %w", err)
	}
	p.ReminderTime = reminder.Format("15:04")
	return p, true, nil
}

// SavePreferences writes the whole set for one person. The service has
// already laid the patch over what was there, so this stores a complete
// picture rather than trying to merge in SQL.
func (r *PostgresRepository) SavePreferences(ctx context.Context, userID uuid.UUID, p Preferences, at time.Time) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `
		INSERT INTO notification_preferences
			(user_id, new_week, prayer_reminder, reminder_time, event_reminders,
			 important_dates, appreciation, journal, goals, challenges, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
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
			updated_at = EXCLUDED.updated_at
	`, userID, p.NewWeek, p.PrayerReminder, p.ReminderTime, p.EventReminders,
		p.ImportantDates, p.Appreciation, p.Journal, p.Goals, p.Challenges, at); err != nil {
		return fmt.Errorf("saving notification preferences: %w", err)
	}
	return nil
}

// Subscribe records a browser, or moves it to this user if it was somebody
// else's. A shared or handed-on device is the reason for that last part: the
// endpoint identifies the browser, and whoever is signed in now is who its
// notifications belong to.
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

// Unsubscribe forgets a browser. The worker calls it when a push service says
// the subscription is gone (FR-NOTF-004); nothing else should need it.
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
