package anniversary

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Couple(ctx context.Context, coupleID uuid.UUID) (start *time.Time, timezone string, err error)
	Letters(ctx context.Context, coupleID uuid.UUID) ([]Letter, error)
	Keep(ctx context.Context, l Letter, at time.Time) (Letter, error)
	Year(ctx context.Context, coupleID uuid.UUID, from, to, today time.Time) (Year, error)
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

// View is the anniversary as one person sees it.
type View struct {
	Occasion Occasion
	// Theirs is the other partner's letter for this occasion, and only once
	// it is open. TheirsSealed says one is waiting without saying what.
	Mine         *Letter
	Theirs       *Letter
	TheirsSealed bool
	// Letters from anniversaries already gone, both of theirs, all open.
	Past []Letter
	Year Year
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (View, error) {
	coupleID, occasion, today, err := s.occasion(ctx, userID)
	if err != nil {
		return View{}, err
	}
	letters, err := s.repo.Letters(ctx, coupleID)
	if err != nil {
		return View{}, err
	}
	year, err := s.repo.Year(ctx, coupleID, occasion.YearBefore(), occasion.On, today)
	if err != nil {
		return View{}, err
	}
	return assemble(userID, occasion, today, letters, year), nil
}

// assemble is where the seal is kept: a letter for this occasion from the
// other partner is only placed in Theirs once it is open.
func assemble(userID uuid.UUID, o Occasion, today time.Time, letters []Letter, year Year) View {
	v := View{Occasion: o, Year: year, Past: []Letter{}}
	for _, l := range letters {
		l := l
		switch {
		case l.OpensOn.Equal(o.On) && l.AuthorID == userID:
			v.Mine = &l
		case l.OpensOn.Equal(o.On):
			if l.Open(today) {
				v.Theirs = &l
			} else {
				v.TheirsSealed = true
			}
		case l.OpensOn.Before(o.On) && l.Open(today):
			v.Past = append(v.Past, l)
		}
	}
	return v
}

// Write keeps this person's letter for the coming anniversary.
func (s *Service) Write(ctx context.Context, userID uuid.UUID, text string) (Letter, error) {
	clean, err := ValidateLetter(text)
	if err != nil {
		return Letter{}, err
	}
	coupleID, occasion, _, err := s.occasion(ctx, userID)
	if err != nil {
		return Letter{}, err
	}
	if !occasion.Writable() {
		return Letter{}, ErrTooEarly
	}
	return s.repo.Keep(ctx, Letter{
		CoupleID: coupleID,
		AuthorID: userID,
		OpensOn:  occasion.On,
		Text:     clean,
	}, s.now())
}

func (s *Service) occasion(ctx context.Context, userID uuid.UUID) (uuid.UUID, Occasion, time.Time, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return uuid.Nil, Occasion{}, time.Time{}, err
	}
	start, timezone, err := s.repo.Couple(ctx, coupleID)
	if err != nil {
		return uuid.Nil, Occasion{}, time.Time{}, err
	}
	today := LocalToday(s.now(), timezone)
	occasion, ok := Next(start, today)
	if !ok {
		return uuid.Nil, Occasion{}, time.Time{}, ErrNoAnniversary
	}
	return coupleID, occasion, today, nil
}
