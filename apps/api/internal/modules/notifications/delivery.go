package notifications

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// DeliveryRepository is what sending one notification needs, whoever is
// sending it.
type DeliveryRepository interface {
	ClaimSend(ctx context.Context, userID uuid.UUID, kind, key string, at time.Time) (bool, error)
	ReleaseSend(ctx context.Context, userID uuid.UUID, kind, key string) error
	SubscriptionsFor(ctx context.Context, userID uuid.UUID) ([]Subscription, error)
	Unsubscribe(ctx context.Context, endpoint string) error
	MarkSent(ctx context.Context, endpoint string, at time.Time) error
	RecordInbox(ctx context.Context, userID uuid.UUID, kind, title, body, path string, at time.Time) error
}

// Deliver claims a notification and sends it to every device that person
// has. The claim is what makes it exactly-once, so it is taken before
// anything is sent and released only if no device took it.
//
// Shared by the worker's tick and by a nudge one partner sends by hand: both
// have to drop subscriptions the push service reports gone (FR-NOTF-004),
// and one copy of that is enough.
func Deliver(
	ctx context.Context,
	repo DeliveryRepository,
	sender push.Sender,
	log *slog.Logger,
	n Notification,
	now time.Time,
) (bool, error) {
	claimed, err := repo.ClaimSend(ctx, n.UserID, n.Kind, n.Key, now)
	if err != nil {
		return false, err
	}
	if !claimed {
		return false, nil
	}

	// The claim is the moment this notification is decided to go out — not
	// whether a device was listening or the push itself worked — so the
	// inbox row is written here, once, regardless of what follows. A held
	// one (quiet hours, the daily cap) never reaches Deliver in the first
	// place, and reaches here — and the inbox — only once it finally does.
	if err := repo.RecordInbox(ctx, n.UserID, n.Kind, n.Message.Title, n.Message.Body, n.Message.Path, now); err != nil {
		log.Warn("could not record a notification in the inbox", "error", err)
	}

	release := func(cause error) error {
		if err := repo.ReleaseSend(ctx, n.UserID, n.Kind, n.Key); err != nil {
			log.Warn("could not release a notification claim", "error", err)
		}
		return cause
	}

	devices, err := repo.SubscriptionsFor(ctx, n.UserID)
	if err != nil {
		return false, release(err)
	}
	if len(devices) == 0 {
		// Claim stands even with no devices — stale by the time they resubscribe.
		return false, nil
	}

	delivered := false
	for _, d := range devices {
		err := sender.Send(ctx, push.Device{Endpoint: d.Endpoint, P256dh: d.P256dh, Auth: d.Auth}, n.Message)
		switch {
		case errors.Is(err, push.ErrGone):
			log.Info("a subscription is gone; removing it", "service", pushService(d.Endpoint))
			if err := repo.Unsubscribe(ctx, d.Endpoint); err != nil {
				log.Warn("could not remove a dead subscription", "error", err)
			}
		case err != nil:
			log.Warn("a device did not take the notification",
				"service", pushService(d.Endpoint), "kind", n.Kind, "error", err)
		default:
			delivered = true
			if err := repo.MarkSent(ctx, d.Endpoint, now); err != nil {
				log.Warn("could not record a send", "error", err)
			}
		}
	}

	if !delivered {
		return false, release(nil)
	}
	return true, nil
}
