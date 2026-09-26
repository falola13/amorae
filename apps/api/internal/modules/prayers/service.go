package prayers

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is what this service needs of storage; declared by the
// consumer, not the implementation.
type Repository interface {
	EnsureWeek(ctx context.Context, coupleID uuid.UUID, weekStart time.Time, setter uuid.UUID, at time.Time) (uuid.UUID, error)
	FirstWeekStart(ctx context.Context, coupleID uuid.UUID) (time.Time, bool, error)
	WeekByID(ctx context.Context, coupleID, weekID uuid.UUID) (Record, error)
	History(ctx context.Context, coupleID uuid.UUID, before time.Time) ([]Record, error)
	WeekOfPoint(ctx context.Context, coupleID, pointID uuid.UUID) (uuid.UUID, error)
	ReplacePoints(ctx context.Context, weekID uuid.UUID, points []Point, at time.Time) error
	Publish(ctx context.Context, weekID, publisherID uuid.UUID, at time.Time) error
	SetCompletion(ctx context.Context, pointID, userID uuid.UUID, prayedOn time.Time, done bool, at time.Time) error
	SetAnswered(ctx context.Context, pointID, userID uuid.UUID, answered bool, note string, at time.Time) error
	Answered(ctx context.Context, coupleID uuid.UUID) ([]Answered, error)
	SetReflection(ctx context.Context, weekID, userID uuid.UUID, body string, at time.Time) error
}

// CoupleContext is everything this module needs about a couple (DEC-27),
// expressed in this package's own types — prayers does not import couples.
type CoupleContext struct {
	CoupleID uuid.UUID
	Location *time.Location
	Members  []Member
	// Today, in the couple's own zone — set by the service (coupleContext),
	// not by Couples.ForPrayers itself, since it depends on the moment of
	// the call, not on couple data. The zero value means "not computed",
	// which ToDTO treats as "not the current week".
	Today time.Time
}

// Partner is the other member.
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

// coupleContext is Couples.ForPrayers plus "today" in the couple's zone,
// computed once here so every caller agrees on what today is, rather than
// each recomputing it from s.now() at a slightly different instant.
func (s *Service) coupleContext(ctx context.Context, userID uuid.UUID) (CoupleContext, error) {
	cc, err := s.couples.ForPrayers(ctx, userID)
	if err != nil {
		return CoupleContext{}, err
	}
	cc.Today = StartOfDay(s.now(), cc.Location)
	return cc, nil
}

// Current is this week, created on first sight if nobody has made it yet.
// No scheduler exists yet (Q-16), so the read creates it; UNIQUE
// (couple_id, week_start) makes concurrent creation safe.
func (s *Service) Current(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error) {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}

	now := s.now()
	weekStart := StartOfWeek(now, cc.Location)

	// Counted from the couple's first week ever, so it doesn't restart when
	// they miss one.
	first, ok, err := s.repo.FirstWeekStart(ctx, cc.CoupleID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if !ok {
		first = weekStart
	}

	setter, err := SetterFor(cc.Members, WeekIndex(first, weekStart))
	if err != nil {
		// Fewer than two members: treat as waiting for partner, not a missing week.
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
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	records, err := s.repo.History(ctx, cc.CoupleID, StartOfWeek(s.now(), cc.Location))
	return records, cc, err
}

// Week is one week by id, and only if it belongs to the caller's couple.
func (s *Service) Week(ctx context.Context, userID, weekID uuid.UUID) (Record, CoupleContext, error) {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// SavePoints replaces this week's points, subject to CanEditPoints.
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

// Publish shares this week with the partner. Publishing twice is a no-op —
// either partner may publish (DEC-33); a caller only reaches here already
// scoped to the couple by Current, so there is nothing further to check.
func (s *Service) Publish(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error) {
	rec, cc, err := s.Current(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	if rec.Status == StatusPublished {
		return rec, cc, nil
	}
	if err := s.repo.Publish(ctx, rec.ID, userID, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, rec.ID)
	return rec, cc, err
}

// SetCompletion marks one point as prayed today, or unmarks it, for the
// caller alone; their partner's progress is never touched. A couple prays
// the week's points every day, not once and done (DEC-33), so this only
// ever touches the current week, and only a point actually scheduled for
// today — anything else is ErrNotForToday rather than silently doing
// something other than what was asked.
func (s *Service) SetCompletion(ctx context.Context, userID, pointID uuid.UUID, done bool) (Record, CoupleContext, error) {
	cc, err := s.coupleContext(ctx, userID)
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
	if rec.Status != StatusPublished {
		return Record{}, CoupleContext{}, ErrNotShared
	}
	if !rec.WeekStart.Equal(StartOfWeek(s.now(), cc.Location)) {
		return Record{}, CoupleContext{}, ErrNotForToday
	}
	point, found := rec.Point(pointID)
	if !found {
		return Record{}, CoupleContext{}, ErrNotFound
	}
	if !ScheduledOn(point.Weekdays, cc.Today.Weekday()) {
		return Record{}, CoupleContext{}, ErrNotForToday
	}

	if err := s.repo.SetCompletion(ctx, pointID, userID, cc.Today, done, s.now()); err != nil {
		return Record{}, CoupleContext{}, err
	}

	rec, err = s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	return rec, cc, err
}

// SetAnswered marks a prayer answered, or takes the mark back. Not
// restricted to the current week — prayers get answered on their own
// schedule. The couple-scoped point lookup is the whole permission check.
func (s *Service) SetAnswered(
	ctx context.Context, userID, pointID uuid.UUID, answered bool, note string,
) (Record, CoupleContext, error) {
	cc, err := s.coupleContext(ctx, userID)
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
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	out, err := s.repo.Answered(ctx, cc.CoupleID)
	if err != nil {
		return nil, CoupleContext{}, err
	}
	return out, cc, nil
}

// SetReflection is the caller's own note on the week — visible to both,
// written by each for themselves, and open regardless of draft/published
// now that a draft is no longer hidden from either partner (DEC-33).
func (s *Service) SetReflection(ctx context.Context, userID, weekID uuid.UUID, body string) (Record, CoupleContext, error) {
	cc, err := s.coupleContext(ctx, userID)
	if err != nil {
		return Record{}, CoupleContext{}, err
	}
	rec, err := s.repo.WeekByID(ctx, cc.CoupleID, weekID)
	if err != nil {
		return Record{}, CoupleContext{}, err
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
