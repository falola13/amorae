package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Event, error)
	ByID(ctx context.Context, coupleID, eventID uuid.UUID) (Event, error)
	Create(ctx context.Context, e Event, at time.Time) (uuid.UUID, error)
	Update(ctx context.Context, e Event, replaceChecklist bool, at time.Time) error
	SetDone(ctx context.Context, coupleID, eventID uuid.UUID, done bool, at time.Time) error
	SetChecklistItem(ctx context.Context, coupleID, eventID, itemID uuid.UUID, done bool, at time.Time) error
	Delete(ctx context.Context, coupleID, eventID uuid.UUID) error
}

// Couples answers which couple is calling; expressed here so events doesn't import the couples package.
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

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

func (s *Service) Get(ctx context.Context, userID, eventID uuid.UUID) (Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	return s.repo.ByID(ctx, coupleID, eventID)
}

// Create returns the whole event so the client never has to re-fetch.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Event{}, err
	}

	e, err := Event{CoupleID: coupleID}.Validate(in, true)
	if err != nil {
		return Event{}, err
	}

	id, err := s.repo.Create(ctx, e, s.now())
	if err != nil {
		return Event{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}

// Update lays input over what's there; the checklist is touched only when the
// request mentions one, so a title-only edit doesn't empty it.
func (s *Service) Update(ctx context.Context, userID, eventID uuid.UUID, in Input) (Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Event{}, err
	}

	current, err := s.repo.ByID(ctx, coupleID, eventID)
	if err != nil {
		return Event{}, err
	}
	updated, err := current.Validate(in, false)
	if err != nil {
		return Event{}, err
	}
	if err := s.repo.Update(ctx, updated, in.Checklist != nil, s.now()); err != nil {
		return Event{}, err
	}
	return s.repo.ByID(ctx, coupleID, eventID)
}

func (s *Service) SetDone(ctx context.Context, userID, eventID uuid.UUID, done bool) (Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if err := s.repo.SetDone(ctx, coupleID, eventID, done, s.now()); err != nil {
		return Event{}, err
	}
	return s.repo.ByID(ctx, coupleID, eventID)
}

func (s *Service) SetChecklistItem(ctx context.Context, userID, eventID, itemID uuid.UUID, done bool) (Event, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Event{}, err
	}
	if err := s.repo.SetChecklistItem(ctx, coupleID, eventID, itemID, done, s.now()); err != nil {
		return Event{}, err
	}
	return s.repo.ByID(ctx, coupleID, eventID)
}

func (s *Service) Delete(ctx context.Context, userID, eventID uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, coupleID, eventID)
}
