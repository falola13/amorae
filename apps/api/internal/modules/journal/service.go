package journal

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Entry, error)
	Create(ctx context.Context, e Entry, at time.Time) (Entry, error)
	Update(ctx context.Context, e Entry) (Entry, error)
	Delete(ctx context.Context, coupleID, authorID, id uuid.UUID) error
}

type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Poker asks the notifications worker to run a pass soon rather than
// waiting for its next cron tick, without journal knowing anything about
// how notifications are put together.
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

// List is the couple's journal, the same for both of them (FR-JRNL-002).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Entry, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Add's author and date never come from the request: author is the session, date is server time.
func (s *Service) Add(ctx context.Context, userID uuid.UUID, tag, text string) (Entry, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Entry{}, err
	}
	cleanTag, cleanText, err := Validate(tag, text)
	if err != nil {
		return Entry{}, err
	}

	// Date isn't set here; the repository derives the couple's local day from the couple row (DEC-27).
	entry, err := s.repo.Create(ctx, Entry{
		CoupleID: coupleID,
		AuthorID: userID,
		Tag:      cleanTag,
		Text:     cleanText,
	}, s.now())
	if err != nil {
		return Entry{}, err
	}
	// A journal entry has no undo window (unlike appreciation), so the
	// partner's notification is due the moment it exists — no reason to
	// make them wait for the next cron tick to hear about it.
	s.poker.Poke()
	return entry, nil
}

// Update touches tag and text only; the date an entry was filed under never
// moves. Scoped by author as well as couple, so editing someone else's entry
// is simply not found, not forbidden.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, tag, text string) (Entry, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Entry{}, err
	}
	cleanTag, cleanText, err := Validate(tag, text)
	if err != nil {
		return Entry{}, err
	}
	return s.repo.Update(ctx, Entry{
		ID:       id,
		CoupleID: coupleID,
		AuthorID: userID,
		Tag:      cleanTag,
		Text:     cleanText,
	})
}

// Delete: only the entry's author may remove it (same scoping as Update).
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, coupleID, userID, id)
}
