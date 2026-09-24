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

// List is every note either of them has sent, newest first. Both see all of
// them: a note one of them cannot read is not a note they were sent.
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
	return s.repo.Create(ctx, Appreciation{
		CoupleID: coupleID,
		FromID:   userID,
		Text:     clean,
	}, s.now())
}

// Undo takes a note back, if it is theirs and the window is still open
// (FR-APPR-003.AC4).
//
// Undoing one that is already gone is not an error: a client that retries
// after a dropped response should find the world as it wanted it, and the
// second delete leaves exactly the same nothing behind.
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
