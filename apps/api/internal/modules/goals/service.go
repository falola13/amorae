package goals

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Goal, error)
	ByID(ctx context.Context, coupleID, goalID uuid.UUID) (Goal, error)
	Create(ctx context.Context, g Goal, at time.Time) (uuid.UUID, error)
	Update(ctx context.Context, g Goal, at time.Time) error
	AddProgress(ctx context.Context, coupleID, goalID, userID uuid.UUID, amount int64, on time.Time) error
}

// Couples answers the one question this module asks of pairing.
type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Service struct {
	repo    Repository
	couples Couples
	now     func() time.Time
}

func NewService(repo Repository, couples Couples, now func() time.Time) *Service {
	return &Service{repo: repo, couples: couples, now: now}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Goal, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

func (s *Service) Get(ctx context.Context, userID, goalID uuid.UUID) (Goal, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Goal{}, err
	}
	return s.repo.ByID(ctx, coupleID, goalID)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Goal, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Goal{}, err
	}
	g, err := Goal{CoupleID: coupleID}.Validate(in, true)
	if err != nil {
		return Goal{}, err
	}
	id, err := s.repo.Create(ctx, g, s.now())
	if err != nil {
		return Goal{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}

func (s *Service) Update(ctx context.Context, userID, goalID uuid.UUID, in Input) (Goal, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Goal{}, err
	}
	current, err := s.repo.ByID(ctx, coupleID, goalID)
	if err != nil {
		return Goal{}, err
	}
	updated, err := current.Validate(in, false)
	if err != nil {
		return Goal{}, err
	}
	if err := s.repo.Update(ctx, updated, s.now()); err != nil {
		return Goal{}, err
	}
	return s.repo.ByID(ctx, coupleID, goalID)
}

// LogProgress appends an entry carrying who logged it and the day it counts
// for. The running total is not stored, so there is nothing else to keep in
// step (BR-GOAL-01).
func (s *Service) LogProgress(ctx context.Context, userID, goalID uuid.UUID, amount int64) (Goal, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Goal{}, err
	}
	if err := ValidateAmount(amount); err != nil {
		return Goal{}, err
	}
	if err := s.repo.AddProgress(ctx, coupleID, goalID, userID, amount, s.now()); err != nil {
		return Goal{}, err
	}
	return s.repo.ByID(ctx, coupleID, goalID)
}
