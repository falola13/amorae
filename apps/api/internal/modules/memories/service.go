package memories

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Memory, error)
	ByID(ctx context.Context, coupleID, id uuid.UUID) (Memory, error)
	Create(ctx context.Context, m Memory, at time.Time) (uuid.UUID, error)
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

// List is the couple's memories, the same for both of them (FR-MEM-002).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Create keeps a moment. Either partner may, and it belongs to them both
// (DEC-16) — an archive with an author beside each entry is a feed.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	clean, err := Validate(in)
	if err != nil {
		return Memory{}, err
	}

	id, err := s.repo.Create(ctx, Memory{
		CoupleID: coupleID,
		Title:    clean.Title,
		Date:     clean.Date,
		Location: clean.Location,
		Note:     clean.Note,
	}, s.now())
	if err != nil {
		return Memory{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}
