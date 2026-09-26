package milestones

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Milestone, error)
	ByID(ctx context.Context, coupleID, id uuid.UUID) (Milestone, error)
	Create(ctx context.Context, m Milestone, at time.Time) (uuid.UUID, error)
	Delete(ctx context.Context, coupleID, id uuid.UUID) error
	// Derived is the anniversary and current members' birthdays, computed
	// from couples and users directly rather than copied in here — see
	// milestones.Milestone's Source field.
	Derived(ctx context.Context, coupleID uuid.UUID) ([]Milestone, error)
}

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

// List is the couple's dates, the same for both of them (FR-DATE-002),
// together with the anniversary and any birthdays derived from the couple
// and its current members. Stored dates are already ordered by date then
// created_at; appending the derived ones and re-sorting by date alone
// (stable, so ties keep that order) gives one list without pretending a
// derived date was created at some particular moment.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Milestone, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	stored, err := s.repo.List(ctx, coupleID)
	if err != nil {
		return nil, err
	}
	derived, err := s.repo.Derived(ctx, coupleID)
	if err != nil {
		return nil, err
	}
	out := append(stored, derived...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

// Create: either partner may add one; no author, it belongs to them both (DEC-16).
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

// Delete refuses a derived id (ErrDerived) rather than reporting it not
// found — it does exist, just not as a row here to remove.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	derived, err := s.repo.Derived(ctx, coupleID)
	if err != nil {
		return err
	}
	for _, d := range derived {
		if d.ID == id {
			return ErrDerived
		}
	}
	return s.repo.Delete(ctx, coupleID, id)
}
