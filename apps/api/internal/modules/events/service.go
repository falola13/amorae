package events

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
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

// Poker asks the notifications worker to run a pass soon rather than
// waiting for its next cron tick, without events knowing anything about how
// notifications are put together.
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
	e.CreatedBy = &userID

	id, err := s.repo.Create(ctx, e, s.now())
	if err != nil {
		return Event{}, err
	}
	// Only a "together" event ever produces a notification (RecentlyWritten
	// excludes "mine" at the query itself) — poking unconditionally costs
	// nothing when there is nothing due.
	s.poker.Poke()
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
	if !mayTouch(current, userID) {
		return Event{}, ErrNotFound
	}
	// Who it is for is the one thing even a "together" event's other
	// partner cannot reassign — everything else about it is theirs equally.
	if in.Kind != nil && current.CreatedBy != nil && *current.CreatedBy != userID {
		return Event{}, apperr.Validation(map[string]string{
			"kind": "Only whoever made it can change who it's for.",
		})
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
	current, err := s.repo.ByID(ctx, coupleID, eventID)
	if err != nil {
		return Event{}, err
	}
	if !mayTouch(current, userID) {
		return Event{}, ErrNotFound
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
	current, err := s.repo.ByID(ctx, coupleID, eventID)
	if err != nil {
		return Event{}, err
	}
	if !mayTouch(current, userID) {
		return Event{}, ErrNotFound
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
	current, err := s.repo.ByID(ctx, coupleID, eventID)
	if err != nil {
		return err
	}
	if !mayTouch(current, userID) {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, coupleID, eventID)
}

// mayTouch reports whether this caller may edit, delete, complete or tick a
// checklist item on this event. A "together" event is either partner's, as
// it always was; a "mine" event is its creator's alone, and a "mine" event
// with no creator on record (there is no such thing going forward, but
// nothing stops one existing) is nobody's in particular, so it is treated
// like "together" rather than locking both partners out.
func mayTouch(e Event, userID uuid.UUID) bool {
	if e.Kind != KindMine {
		return true
	}
	return e.CreatedBy == nil || *e.CreatedBy == userID
}
