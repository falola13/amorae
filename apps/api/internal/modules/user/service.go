package user

import (
	"context"
	"time"

	"github.com/google/uuid"
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

type UpdateProfileInput struct {
	DisplayName string `json:"display_name"`
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	displayName, err := ValidateDisplayName(input.DisplayName)
	if err != nil {
		return User{}, err
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	u.DisplayName = displayName
	u.UpdatedAt = s.now()

	return s.repo.Update(ctx, u)
}
