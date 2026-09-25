package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Repository lists only what Service calls; Create lives on the concrete
// type instead, for auth's use (see PostgresRepository's ISP note).
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

// Email is intentionally absent: changing it needs the current password
// (auth.Service.ChangeEmail).
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

// collectFields merges a validation error's fields so multiple invalid fields are reported together.
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
