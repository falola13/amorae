package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Repository is declared here, by the consumer (Service), not by whatever
// implements it — that's what lets repository_postgres.go depend on this
// package instead of the other way around. It only lists what Service
// actually calls; Create lives on the concrete type for auth's benefit, not
// here (see the ISP note on auth.UserRepository).
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	Update(ctx context.Context, u User) (User, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateProfileInput holds the non-sensitive profile fields. Email is not
// here on purpose: changing it needs the current password, so it's its own
// use case (auth.Service.ChangeEmail).
type UpdateProfileInput struct {
	DisplayName string `json:"display_name"`
	// Timezone is optional: empty leaves it unchanged.
	Timezone string `json:"timezone"`
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	fields := map[string]string{}

	displayName, err := ValidateDisplayName(input.DisplayName)
	if err := collectFields(fields, err); err != nil {
		return User{}, err
	}

	var timezone string
	if input.Timezone != "" {
		timezone, err = ValidateTimezone(input.Timezone)
		if err := collectFields(fields, err); err != nil {
			return User{}, err
		}
	}

	if len(fields) > 0 {
		return User{}, apperr.Validation(fields)
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	u.DisplayName = displayName
	if timezone != "" {
		u.Timezone = timezone
	}
	u.UpdatedAt = s.now()

	return s.repo.Update(ctx, u)
}

// collectFields merges a validation error's field messages into fields, so
// every invalid field is reported at once. Any other error is returned as is.
func collectFields(fields map[string]string, err error) error {
	if err == nil {
		return nil
	}
	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindInvalid {
		return err
	}
	for k, v := range appErr.Fields {
		fields[k] = v
	}
	return nil
}
