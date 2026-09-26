package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/push"
)

// Repository is what this service needs of storage, named by the consumer.
type Repository interface {
	PreferencesFor(ctx context.Context, userID uuid.UUID) (Preferences, bool, error)
	SavePreferences(ctx context.Context, userID uuid.UUID, p Preferences, at time.Time) error
	Subscribe(ctx context.Context, sub Subscription, at time.Time) error

	// For a nudge, which is sent from a request rather than from the tick.
	DeliveryRepository
	InboxRepository
	NudgeTarget(ctx context.Context, senderID uuid.UUID) (uuid.UUID, string, error)
	BudgetFor(ctx context.Context, userID uuid.UUID, now time.Time) (Budget, error)
	CountSends(ctx context.Context, userID uuid.UUID, kind string, since time.Time) (int, error)
}

type Service struct {
	repo   Repository
	sender push.Sender
	log    *slog.Logger
	now    func() time.Time
}

func NewService(repo Repository, sender push.Sender, log *slog.Logger, now func() time.Time) *Service {
	return &Service{repo: repo, sender: sender, log: log, now: now}
}

// Get returns this person's settings, or the defaults if unset. Reading
// does not create a row.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Preferences, error) {
	p, _, err := s.repo.PreferencesFor(ctx, userID)
	return p, err
}

// Update lays a patch over what is there and answers with the whole set, so
// the client never has to re-read to see what it just changed.
func (s *Service) Update(ctx context.Context, userID uuid.UUID, patch Patch) (Preferences, error) {
	current, _, err := s.repo.PreferencesFor(ctx, userID)
	if err != nil {
		return Preferences{}, err
	}
	updated, err := current.Apply(patch)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.SavePreferences(ctx, userID, updated, s.now()); err != nil {
		return Preferences{}, err
	}
	return updated, nil
}

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error {
	endpoint, p256dh, auth, err := ValidateSubscription(endpoint, p256dh, auth)
	if err != nil {
		return err
	}
	return s.repo.Subscribe(ctx, Subscription{
		UserID:   userID,
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
	}, s.now())
}
