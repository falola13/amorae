package prayers

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is what this service needs of storage, declared here rather than
// next to the implementation: the consumer names the contract, so the
// repository can grow methods nobody here has to know about.
type Repository interface {
	EnsureWeek(ctx context.Context, coupleID uuid.UUID, weekStart time.Time, setter uuid.UUID, at time.Time) (uuid.UUID, error)
	FirstWeekStart(ctx context.Context, coupleID uuid.UUID) (time.Time, bool, error)
	WeekByID(ctx context.Context, coupleID, weekID uuid.UUID) (Record, error)
	History(ctx context.Context, coupleID uuid.UUID, before time.Time) ([]Record, error)
	WeekOfPoint(ctx context.Context, coupleID, pointID uuid.UUID) (uuid.UUID, error)
	ReplacePoints(ctx context.Context, weekID uuid.UUID, points []Point, at time.Time) error
	Publish(ctx context.Context, weekID uuid.UUID, at time.Time) error
	SetCompletion(ctx context.Context, pointID, userID uuid.UUID, done bool, at time.Time) error
	SetAnswered(ctx context.Context, pointID, userID uuid.UUID, answered bool, note string, at time.Time) error
	Answered(ctx context.Context, coupleID uuid.UUID) ([]Answered, error)
	SetReflection(ctx context.Context, weekID, userID uuid.UUID, body string, at time.Time) error
}

// CoupleContext is everything this module needs to know about a couple, and
// nothing else: which couple, whose turn it can be, and the zone its week
// turns over in (DEC-27).
//
// It is expressed in this package's own types on purpose. Prayers does not
// import couples; the composition root adapts one to the other, so neither
// module knows the other exists.
type CoupleContext struct {
	CoupleID uuid.UUID
	Location *time.Location
	Members  []Member
}

// Partner is the other member, for a view that has to separate "mine" from
// "theirs".
func (cc CoupleContext) Partner(userID uuid.UUID) uuid.UUID {
	for _, m := range cc.Members {
		if m.UserID != userID {
			return m.UserID
		}
	}
	return uuid.UUID{}
}

// Couples answers the one question this module asks of pairing.
type Couples interface {
	ForPrayers(ctx context.Context, userID uuid.UUID) (CoupleContext, error)
}

type Service struct {
	repo    Repository
	couples Couples
	now     func() time.Time
}

func NewService(repo Repository, couples Couples, now func() time.Time) *Service {
	return &Service{repo: repo, couples: couples, now: now}
}

// Current is this week, created on first sight if nobody has made it yet.
//
// The contract once said a scheduler would create weeks and the API never
// would. There is no scheduler yet (Q-16), and waiting for one would mean the
// feature does not work at all — so the read creates it, which is safe for
// exactly the reason the scheduler would have been: UNIQUE (couple_id,
// week_start) means the second writer loses harmlessly. When the worker
// arrives it pre-warms the week and sends the notification; it does not
// become a prerequisite.
func (s *Service) Current(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	now := s.now()
	weekStart := StartOfWeek(now, cc.Location)

	// The rotation is counted from the couple's first week ever, so it does
	// not restart when they miss one.
	first, ok, err := s.repo.FirstWeekStart(ctx, cc.CoupleID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if !ok {
		first = weekStart
	}

	setter, err := SetterFor(cc.Members, WeekIndex(first, weekStart))
	if err != nil {
		// One member, or none: there is nobody whose turn it could be. That
		// is the couple's state, not a missing week, and it reads very
		// differently to someone waiting for their partner to join.
		return Record{}, CoupleContext{}, ErrWaitingForPartner
	}

	weekID, err := s.repo.EnsureWeek(ctx, cc.CoupleID, weekStart, setter, now)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// History is every week before this one, newest first.
func (s *Service) History(ctx context.Context, userID uuid.UUID) ([]Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	records, err := s.repo.History(ctx, cc.CoupleID, StartOfWeek(s.now(), cc.Location))
	return records, cc, err
}

// Week is one week by id, and only if it belongs to the caller's couple.
func (s *Service) Week(ctx context.Context, userID, weekID uuid.UUID) (Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// SavePoints replaces this week's points. The setter may keep editing all
// week — fixing a typo is not a betrayal, and a week you cannot add to on
// Wednesday is a week that stops being useful on Monday. What they cannot do
// is rewrite or remove a prayer their partner has already prayed; see
// CanEditPoints.
func (s *Service) SavePoints(ctx context.Context, userID uuid.UUID, points []Point) (Record, CoupleContext, error) {
	rec, cc, err := s.Current(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	cleaned, err := ValidatePoints(points)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if err := CanEditPoints(rec.Week, userID, cleaned, rec.PrayedByOthers(userID)); err != nil {
		return Record{}, CoupleContext{}, err
	}
	if err := s.repo.ReplacePoints(ctx, rec.ID, cleaned, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, rec.ID)
	return rec, cc, err
}

// Publish shares this week with the partner. Publishing twice is not an
// error: a client that retries should find the world as it wanted it.
func (s *Service) Publish(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error) {
	rec, cc, err := s.Current(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if err := CanPublish(rec.Week, userID); err != nil {
		return Record{}, CoupleContext{}, err
	}
	if rec.Status == StatusPublished {
		return rec, cc, nil
	}
	if err := s.repo.Publish(ctx, rec.ID, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, rec.ID)
	return rec, cc, err
}

// SetCompletion marks one point as prayed, or unmarks it, for the caller
// alone. Their partner's progress is never touched — that is the whole model.
func (s *Service) SetCompletion(ctx context.Context, userID, pointID uuid.UUID, done bool) (Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	// Scoped to the couple, so a point of somebody else's is simply not there.
	weekID, err := s.repo.WeekOfPoint(ctx, cc.CoupleID, pointID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	// A week still being written is invisible to the partner, so there is
	// nothing there for them to have prayed.
	if StatusFor(rec.Week, userID) == StatusWaiting {
		return Record{}, CoupleContext{}, ErrNotFound
	}

	if err := s.repo.SetCompletion(ctx, pointID, userID, done, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// SetReflection stores the caller's own words about a week. Both partners
// write their own, and both can read both.
// SetAnswered marks a prayer answered, or takes the mark back.
//
// Deliberately not restricted to the current week. Prayers are answered on
// their own schedule — months later, long after the week has closed into
// history — and a feature that only worked for seven days would miss most of
// what it exists to catch. The couple-scoped lookup of the point is the whole
// permission check: a point belonging to anyone else is simply not found.
func (s *Service) SetAnswered(
	ctx context.Context, userID, pointID uuid.UUID, answered bool, note string,
) (Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	note, err = ValidateAnswerNote(note)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	weekID, err := s.repo.WeekOfPoint(ctx, cc.CoupleID, pointID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if err := CanAnswer(rec.Week); err != nil {
		return Record{}, CoupleContext{}, err
	}

	if err := s.repo.SetAnswered(ctx, pointID, userID, answered, note, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// Answered is everything the couple has marked answered, newest first.
func (s *Service) Answered(ctx context.Context, userID uuid.UUID) ([]Answered, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	out, err := s.repo.Answered(ctx, cc.CoupleID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	return out, cc, nil
}

func (s *Service) SetReflection(ctx context.Context, userID, weekID uuid.UUID, body string) (Record, CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if StatusFor(rec.Week, userID) == StatusWaiting {
		return Record{}, CoupleContext{}, ErrNotFound
	}

	body, err = ValidateReflection(body)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if err := s.repo.SetReflection(ctx, weekID, userID, body, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}
