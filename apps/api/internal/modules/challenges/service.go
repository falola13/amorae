package challenges

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type Repository interface {
	// Latest is the couple's newest challenge of any status, read as of
	// `now` in their own zone; when any are active, the most recently started
	// of those. ErrNotFound when they have never had one.
	Latest(ctx context.Context, coupleID uuid.UUID, now time.Time) (Challenge, error)
	// Active is every challenge the couple has going, oldest started first.
	Active(ctx context.Context, coupleID uuid.UUID, now time.Time) ([]Challenge, error)
	// Get is any one of the couple's challenges; ErrNoSuchChallenge otherwise.
	Get(ctx context.Context, coupleID, id uuid.UUID, now time.Time) (Challenge, error)
	// Past is the couple's finished and left challenges, newest first, with
	// the caller's count and their partner's.
	Past(ctx context.Context, coupleID, userID uuid.UUID) ([]Summary, error)
	// Today is the couple's own date as of `now`.
	Today(ctx context.Context, coupleID uuid.UUID, now time.Time) (time.Time, error)
	// Start begins a challenge unless the starter already has MaxActive going,
	// or the partner does and it is shared (ErrTooMany), or the same curated
	// one is already running for them (ErrAlreadyRunning).
	Start(ctx context.Context, coupleID, createdBy uuid.UUID, s Spec, now time.Time) (uuid.UUID, error)
	// Edit changes the title, start day or kind of an active challenge the
	// caller may touch, by the rules in Challenge.CheckEdit.
	Edit(ctx context.Context, coupleID, userID, id uuid.UUID, e Edit, now time.Time) error
	// ReplacePlan sets the days' texts by position, by the rules in
	// Challenge.CheckPlan, finishing it if that leaves every day answered.
	ReplacePlan(ctx context.Context, coupleID, userID, id uuid.UUID, prompts []string, at time.Time) error
	// Record writes one person's entry for one day of an active challenge
	// and, in the same write, finishes the challenge if that made every
	// member's every day marked. ErrOver when it is no longer active.
	Record(ctx context.Context, coupleID, userID, id uuid.UUID, n int, e Entry, at time.Time) error
	// End leaves an active challenge, keeping it; ErrNotFound when it is not
	// one that is going.
	End(ctx context.Context, coupleID, userID, id uuid.UUID, at time.Time) error
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

// CanEdit is whether this person can change it: it is still going and it is
// theirs to touch.
func (v Viewer) CanEdit() bool { return v.Status == StatusActive && v.MayTouch(v.Me) }

// Active is everything the couple has going, oldest started first; empty
// when there is nothing.
func (s *Service) Active(ctx context.Context, userID uuid.UUID) ([]Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.Active(ctx, coupleID, s.now())
	if err != nil {
		return nil, err
	}
	out := make([]Viewer, 0, len(list))
	for _, c := range list {
		out = append(out, Viewer{Challenge: c, Me: userID})
	}
	return out, nil
}

// Current is the most recently started challenge that is going; a couple
// with none — even one who has just finished one — gets ErrNotFound. It is
// what the older, single-challenge routes act on.
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

// StartInput is a curated template's key, a challenge the couple wrote, or an
// earlier challenge to do again; exactly one of the three. Kind and StartedOn
// are optional: empty is together, and today.
type StartInput struct {
	Template  string
	Custom    *Custom
	Again     string
	Kind      string
	StartedOn string
}

func (s *Service) Start(ctx context.Context, userID uuid.UUID, in StartInput) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	kind, err := ValidateKind(in.Kind)
	if err != nil {
		return Viewer{}, err
	}
	var startedOn *time.Time
	if in.StartedOn != "" {
		on, err := ParseDay(in.StartedOn)
		if err != nil {
			return Viewer{}, err
		}
		today, err := s.repo.Today(ctx, coupleID, s.now())
		if err != nil {
			return Viewer{}, err
		}
		if err := ValidateStart(today, on); err != nil {
			return Viewer{}, err
		}
		startedOn = &on
	}

	var t Template
	if in.Again != "" {
		if in.Template != "" || in.Custom != nil {
			return Viewer{}, apperr.Validation(map[string]string{"again": "Choose one of ours, write your own, or do one again."})
		}
		again, err := uuid.Parse(in.Again)
		if err != nil {
			return Viewer{}, ErrNoSuchChallenge
		}
		before, err := s.repo.Get(ctx, coupleID, again, s.now())
		if err != nil {
			return Viewer{}, err
		}
		t = Template{Key: before.Template, Title: before.Title}
		for _, d := range before.Days {
			t.Prompts = append(t.Prompts, d.Prompt)
		}
		// Somebody else's own stays theirs; doing it again is a shared one
		// unless the person asking is who it belonged to — and said so.
		if in.Kind == "" && before.Kind == KindMine && before.CreatedBy == userID {
			kind = KindMine
		}
	} else {
		t, err = templateFor(in)
		if err != nil {
			return Viewer{}, err
		}
	}

	id, err := s.repo.Start(ctx, coupleID, userID, Spec{Template: t, Kind: kind, StartedOn: startedOn}, s.now())
	if err != nil {
		return Viewer{}, err
	}
	// The other partner is told it was started; sooner than the next tick.
	s.poker.Poke()
	return s.Get(ctx, userID, id)
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
func (s *Service) Mark(ctx context.Context, userID, id uuid.UUID, n int, done, skipped *bool, note *string) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}

	entry, err := ValidateEntry(done, skipped, note)
	if err != nil {
		return Viewer{}, err
	}

	c, err := s.repo.Get(ctx, coupleID, id, s.now())
	if err != nil {
		return Viewer{}, err
	}
	// A "just me" challenge is its creator's to write on, and nobody else's.
	if !c.MayTouch(userID) {
		return Viewer{}, ErrNoSuchChallenge
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

	if err := s.repo.Record(ctx, coupleID, userID, c.ID, n, entry, s.now()); err != nil {
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

// Leave ends one active challenge but keeps it — its days, marks and notes —
// so it can be looked back on and something new can be started. One that is
// already over is not there to leave: ErrNotFound.
func (s *Service) Leave(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.End(ctx, coupleID, userID, id, s.now())
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
	if !c.MayTouch(userID) {
		return Viewer{}, ErrNoSuchChallenge
	}
	if c.Status == StatusActive {
		return Viewer{}, ErrNotOver
	}
	if err := s.repo.SetReflection(ctx, coupleID, userID, id, text, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Get(ctx, userID, id)
}

// EditInput is what a client sent to change; a nil field is left alone.
type EditInput struct {
	Title     *string
	StartedOn *string
	Kind      *string
}

// Edit changes an active challenge's title, start day or kind. Who may, and
// what is still allowed once it has begun or been joined, is Challenge.CheckEdit.
func (s *Service) Edit(ctx context.Context, userID, id uuid.UUID, in EditInput) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	e := Edit{Title: in.Title}
	if in.StartedOn != nil {
		on, err := ParseDay(*in.StartedOn)
		if err != nil {
			return Viewer{}, err
		}
		e.StartedOn = &on
	}
	if in.Kind != nil {
		if *in.Kind == "" {
			return Viewer{}, apperr.Validation(map[string]string{"kind": "Choose together or just me."})
		}
		kind, err := ValidateKind(*in.Kind)
		if err != nil {
			return Viewer{}, err
		}
		e.Kind = &kind
	}
	e, err = ValidateEdit(e)
	if err != nil {
		return Viewer{}, err
	}
	if err := s.repo.Edit(ctx, coupleID, userID, id, e, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Get(ctx, userID, id)
}

// ReplacePlan sets what each day says, by position, extending or shortening
// the challenge. Returns it as the caller now sees it, finished if that was
// the last thing standing between the participants and the end.
func (s *Service) ReplacePlan(ctx context.Context, userID, id uuid.UUID, prompts []string) (Viewer, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	prompts, err = ValidatePlan(prompts)
	if err != nil {
		return Viewer{}, err
	}
	if err := s.repo.ReplacePlan(ctx, coupleID, userID, id, prompts, s.now()); err != nil {
		return Viewer{}, err
	}
	return s.Get(ctx, userID, id)
}
