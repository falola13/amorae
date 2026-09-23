package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is what this service needs of storage, named by the consumer.
type Repository interface {
	PreferencesFor(ctx context.Context, userID uuid.UUID) (Preferences, bool, error)
	SavePreferences(ctx context.Context, userID uuid.UUID, p Preferences, at time.Time) error
	Subscribe(ctx context.Context, sub Subscription, at time.Time) error
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

// Get is this person's settings, or the defaults if they have never changed
// any. Reading does not create a row: somebody who never opens the screen
// should leave no trace of having been asked.
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

// Subscribe records a browser against this person.
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
