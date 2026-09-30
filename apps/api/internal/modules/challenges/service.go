package challenges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type Repository interface {
	// Latest is the couple's newest challenge of any status, read as of
	// `now` in their own zone; the active one, when there is one, is always
	// the newest. ErrNotFound when they have never had one.
	Latest(ctx context.Context, coupleID uuid.UUID, now time.Time) (Challenge, error)
	// Get is any one of the couple's challenges; ErrNoSuchChallenge otherwise.
	Get(ctx context.Context, coupleID, id uuid.UUID, now time.Time) (Challenge, error)
	// Past is the couple's finished and left challenges, newest first, with
	// the caller's count and their partner's.
	Past(ctx context.Context, coupleID, userID uuid.UUID) ([]Summary, error)
	Start(ctx context.Context, coupleID, createdBy uuid.UUID, t Template, on time.Time) (uuid.UUID, error)
	// Record writes one person's entry for one day of the active challenge
	// and, in the same write, finishes the challenge if that made every
	// member's every day marked. ErrOver when it is no longer active.
	Record(ctx context.Context, coupleID, userID uuid.UUID, n int, e Entry, at time.Time) error
	// End leaves the active challenge, keeping it; ErrNotFound when none is.
	End(ctx context.Context, coupleID uuid.UUID, at time.Time) error
	// SetReflection saves, or with "" removes, one person's reflection on a
	// challenge that is over.
	SetReflection(ctx context.Context, coupleID, userID, id uuid.UUID, text string, at time.Time) error
}

type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Poker asks the notifications worker to run a pass soon rather than
// waiting for its next cron tick, without challenges knowing anything about
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

// Viewer is a challenge as one partner sees it: their marks and their
// partner's alongside — visible, not editable by the other.
type Viewer struct {
	Challenge
	Me      uuid.UUID
	Partner uuid.UUID
}

// Current is the challenge that is going; a couple with none — even one who
// has just finished one — gets ErrNotFound.
func (s *Service) Current(ctx context.Context, userID uuid.UUID) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	c, err := s.repo.Latest(ctx, coupleID, s.now())
	if err != nil {
		return Viewer{}, err
	}
	if c.Status != StatusActive {
		return Viewer{}, ErrNotFound
	}
	return Viewer{Challenge: c, Me: userID}, nil
}

// Get is any of the couple's challenges, going or kept.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	c, err := s.repo.Get(ctx, coupleID, id, s.now())
	if err != nil {
		return Viewer{}, err
	}
	return Viewer{Challenge: c, Me: userID}, nil
}

func (s *Service) Past(ctx context.Context, userID uuid.UUID) ([]Summary, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.Past(ctx, coupleID, userID)
}

// StartInput is either a curated template's key or a challenge the couple
// wrote; never both.
type StartInput struct {
	Template string
	Custom   *Custom
}

func (s *Service) Start(ctx context.Context, userID uuid.UUID, in StartInput) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	t, err := templateFor(in)
	if err != nil {
		return Viewer{}, err
	}
	if _, err := s.repo.Start(ctx, coupleID, userID, t, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Current(ctx, userID)
}

func templateFor(in StartInput) (Template, error) {
	if in.Custom == nil {
		return TemplateByKey(in.Template)
	}
	if in.Template != "" {
		return Template{}, apperr.Validation(map[string]string{"template": "Choose one of ours or write your own, not both."})
	}
	return ValidateCustom(*in.Custom)
}

// Mark records what this partner says about a day, and only theirs (DEC-30):
// a mark, a note, or both. Days open on the calendar, one a day, but a day
// already open stays open.
func (s *Service) Mark(ctx context.Context, userID uuid.UUID, n int, done, skipped *bool, note *string) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}

	entry, err := ValidateEntry(done, skipped, note)
	if err != nil {
		return Viewer{}, err
	}

	c, err := s.repo.Latest(ctx, coupleID, s.now())
	if err != nil {
		return Viewer{}, err
	}
	if c.Status != StatusActive {
		return Viewer{}, ErrOver
	}
	if n < 1 || n > len(c.Days) {
		return Viewer{}, ErrUnknownDay
	}
	if !c.Opened(n) {
		return Viewer{}, ErrDayNotOpen
	}

	if err := s.repo.Record(ctx, coupleID, userID, n, entry, s.now()); err != nil {
		return Viewer{}, err
	}
	if entry.SetMark {
		// Only a mark, never a clear, can be the one that makes it "both of
		// you" (ForBothMarked) — worth telling them sooner than the next
		// cron tick.
		s.poker.Poke()
	}
	return s.Get(ctx, userID, c.ID)
}

// Leave ends the active challenge but keeps it — its days, marks and notes —
// so it can be looked back on and something new can be started.
func (s *Service) Leave(ctx context.Context, userID uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.End(ctx, coupleID, s.now())
}

// Reflect saves what this person took from a challenge that is over; empty
// text takes it back. Returns the challenge as they now see it.
func (s *Service) Reflect(ctx context.Context, userID, id uuid.UUID, text string) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	text, err = ValidateReflection(text)
	if err != nil {
		return Viewer{}, err
	}
	c, err := s.repo.Get(ctx, coupleID, id, s.now())
	if err != nil {
		return Viewer{}, err
	}
	if c.Status == StatusActive {
		return Viewer{}, ErrNotOver
	}
	if err := s.repo.SetReflection(ctx, coupleID, userID, id, text, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Get(ctx, userID, id)
}
