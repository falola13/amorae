package journal

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Entry, error)
	Create(ctx context.Context, e Entry, at time.Time) (Entry, error)
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

// List is the couple's journal, the same for both of them (FR-JRNL-002).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Entry, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Add writes an entry, dated today and authored by whoever sent it. Neither
// of those comes from the request: the author is who the session says it is,
// and the date is when it was written.
func (s *Service) Add(ctx context.Context, userID uuid.UUID, tag, text string) (Entry, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Entry{}, err
	}
	cleanTag, cleanText, err := Validate(tag, text)
	if err != nil {
		return Entry{}, err
	}

	// The date is not set here: it is the couple's local day, which the
	// repository works out from the couple row (DEC-27).
	return s.repo.Create(ctx, Entry{
		CoupleID: coupleID,
		AuthorID: userID,
		Tag:      cleanTag,
		Text:     cleanText,
	}, s.now())
}
