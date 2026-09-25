package milestones

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Milestone, error)
	ByID(ctx context.Context, coupleID, id uuid.UUID) (Milestone, error)
	Create(ctx context.Context, m Milestone, at time.Time) (uuid.UUID, error)
	Delete(ctx context.Context, coupleID, id uuid.UUID) error
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

// List is the couple's dates, the same for both of them (FR-DATE-002).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Milestone, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Create keeps a date. Either partner may add one and it belongs to them both
// (DEC-16), so there is no author on it — a date is not an opinion.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Milestone, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Milestone{}, err
	}
	clean, err := Validate(in)
	if err != nil {
		return Milestone{}, err
	}

	id, err := s.repo.Create(ctx, Milestone{
		CoupleID: coupleID,
		Title:    clean.Title,
		Date:     clean.Date,
		Sub:      clean.Sub,
		Reminder: clean.Reminder,
	}, s.now())
	if err != nil {
		return Milestone{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, coupleID, id)
}
