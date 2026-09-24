package challenges

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Current(ctx context.Context, coupleID uuid.UUID) (Challenge, error)
	Start(ctx context.Context, coupleID uuid.UUID, t Template, on time.Time) (uuid.UUID, error)
	SetMark(ctx context.Context, coupleID, userID uuid.UUID, n int, mark Mark, at time.Time) error
	ClearMark(ctx context.Context, coupleID, userID uuid.UUID, n int) error
	Leave(ctx context.Context, coupleID uuid.UUID) error
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

// Viewer is a challenge as one partner sees it: their own marks, and their
// partner's alongside. Both see both, as with prayer completion — seeing is
// not the same as being able to change.
type Viewer struct {
	Challenge
	Me      uuid.UUID
	Partner uuid.UUID
}

func (s *Service) Current(ctx context.Context, userID uuid.UUID) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	c, err := s.repo.Current(ctx, coupleID)
	if err != nil {
		return Viewer{}, err
	}
	return Viewer{Challenge: c, Me: userID}, nil
}

func (s *Service) Start(ctx context.Context, userID uuid.UUID, key string) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	t, err := TemplateByKey(key)
	if err != nil {
		return Viewer{}, err
	}
	if _, err := s.repo.Start(ctx, coupleID, t, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Current(ctx, userID)
}

// Mark records what this partner says about a day, and only theirs (DEC-30).
func (s *Service) Mark(ctx context.Context, userID uuid.UUID, n int, done, skipped *bool) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}

	mark, marked, err := ValidateMark(done, skipped)
	if err != nil {
		return Viewer{}, err
	}
	if marked {
		err = s.repo.SetMark(ctx, coupleID, userID, n, mark, s.now())
	} else {
		err = s.repo.ClearMark(ctx, coupleID, userID, n)
	}
	if err != nil {
		return Viewer{}, err
	}
	return s.Current(ctx, userID)
}

func (s *Service) Leave(ctx context.Context, userID uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Leave(ctx, coupleID)
}
