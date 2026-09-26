package appreciation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Appreciation, error)
	ByID(ctx context.Context, coupleID, id uuid.UUID) (Appreciation, error)
	Create(ctx context.Context, a Appreciation, at time.Time) (Appreciation, error)
	Delete(ctx context.Context, coupleID, id uuid.UUID) error
}

type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Poker asks the notifications worker to run a pass soon rather than
// waiting for its next cron tick, without appreciation knowing anything
// about how notifications are put together.
type Poker interface{ Poke() }

type Service struct {
	repo    Repository
	couples Couples
	now     func() time.Time
	poker   Poker
}

func NewService(repo Repository, couples Couples, now func() time.Time, poker Poker) *Service {
	return &Service{repo: repo, couples: couples, now: now, poker: poker}
}

// List returns every note either of them sent, newest first; both partners see all of them.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Appreciation, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Send writes a note from this person to the other one (FR-APPR-001).
func (s *Service) Send(ctx context.Context, userID uuid.UUID, text string) (Appreciation, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Appreciation{}, err
	}
	clean, err := ValidateText(text)
	if err != nil {
		return Appreciation{}, err
	}
	a, err := s.repo.Create(ctx, Appreciation{
		CoupleID: coupleID,
		FromID:   userID,
		Text:     clean,
	}, s.now())
	if err != nil {
		return Appreciation{}, err
	}
	// The poke just runs a tick now instead of at the next cron one; the
	// notification itself still waits out CanUndo's window before it can
	// go (worker_repository.go's appreciationUndoWindow), so a poke here
	// changes nothing about whether an undone note gets announced.
	s.poker.Poke()
	return a, nil
}

// Undo takes back an owned note within the window (FR-APPR-003.AC4).
// Already-gone is not an error, so a retried request is idempotent.
func (s *Service) Undo(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	a, err := s.repo.ByID(ctx, coupleID, id)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := CanUndo(a, userID, s.now()); err != nil {
		return err
	}
	return s.repo.Delete(ctx, coupleID, id)
}
