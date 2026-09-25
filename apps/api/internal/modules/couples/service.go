package couples

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type Repository interface {
	Create(ctx context.Context, inviteCode string, inviteExpiresAt time.Time, input COUPLES) (COUPLES, error)
	Join(ctx context.Context, userID uuid.UUID, code string, at time.Time) error
	GetForUser(ctx context.Context, userID uuid.UUID, now time.Time) (Mine, error)
	UpdateCouples(ctx context.Context, coupleID uuid.UUID, start *time.Time, name, timezone *string) error
	UpdateRole(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, role string) error
	UpdateOnboarding(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, install *bool, notifications *bool) error
	ReplaceInvite(ctx context.Context, userID uuid.UUID, code string, expiresAt, at time.Time) error
	Dissolve(ctx context.Context, userID uuid.UUID, at time.Time) error
	GetArchivedForUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]Mine, error)
}

// AttemptLimiter caps join attempts per person against brute-forcing invite codes.
type AttemptLimiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// Events counts product events for metrics — counts only, never content.
type Events interface {
	CoupleCreated()
	CouplePaired()
	CoupleEnded()
}

type noEvents struct{}

func (noEvents) CoupleCreated() {}
func (noEvents) CouplePaired()  {}
func (noEvents) CoupleEnded()   {}

const inviteTTL = 7 * 24 * time.Hour

// With 17.6 million possible codes, three draws make a repeat practically impossible.
const inviteCodeTries = 3

type Service struct {
	repo     Repository
	now      func() time.Time
	attempts AttemptLimiter
	events   Events
}

// events may be nil, which counts nothing.
func NewService(repo Repository, now func() time.Time, attempts AttemptLimiter, events Events) *Service {
	if events == nil {
		events = noEvents{}
	}
	return &Service{repo: repo, now: now, attempts: attempts, events: events}
}

type CoupleCreateInput struct {
	Name                  *string
	RelationshipStartDate *string
	// Seeds the couple's zone; nothing lets it be changed later.
	Timezone string
}

func parseDay(day *string) (*time.Time, error) {
	if day == nil {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*day))
	if err != nil {
		return nil, apperr.Validation(map[string]string{
			"relationship_start_date": "Use a date like 2025-10-25.",
		})
	}
	return &parsed, nil
}

func (s *Service) newInviteCode() (string, error) {
	letters := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits := []byte("0123456789")
	code := make([]byte, 6)

	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		code[i] = letters[n.Int64()]
	}
	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[3+i] = digits[n.Int64()]
	}
	return string(code), nil
}

func (s *Service) Create(ctx context.Context, createdBy uuid.UUID, creatorName string, input CoupleCreateInput) (COUPLES, error) {
	name := ""
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
	}
	if name == "" {
		name = strings.TrimSpace(creatorName) + " & partner"
	}

	start, err := parseDay(input.RelationshipStartDate)
	if err != nil {
		return COUPLES{}, err
	}

	timezone := input.Timezone
	if strings.TrimSpace(timezone) == "" {
		timezone = user.DefaultTimezone
	}

	now := s.now()
	value, err := New(name, createdBy, timezone, start, now)
	if err != nil {
		return COUPLES{}, err
	}

	var created COUPLES
	err = s.withFreshCode(func(code string) error {
		created, err = s.repo.Create(ctx, code, now.Add(inviteTTL), value)
		return err
	})
	if err != nil {
		return COUPLES{}, err
	}
	s.events.CoupleCreated()
	return created, nil
}

// withFreshCode draws a new code again when the repository reports it taken;
// each try is its own transaction since a failed insert aborts the one it ran in.
func (s *Service) withFreshCode(write func(code string) error) error {
	for try := 0; ; try++ {
		code, err := s.newInviteCode()
		if err != nil {
			return apperr.Internal(err)
		}
		err = write(code)
		if !errors.Is(err, errCodeTaken) || try == inviteCodeTries-1 {
			return err
		}
	}
}

// RegenerateInvite's old code stops working at once; a full couple gets ErrCoupleFull.
func (s *Service) RegenerateInvite(ctx context.Context, userID uuid.UUID) (Mine, error) {
	now := s.now()
	if err := s.withFreshCode(func(code string) error {
		return s.repo.ReplaceInvite(ctx, userID, code, now.Add(inviteTTL), now)
	}); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) Join(ctx context.Context, userID uuid.UUID, code string) (Mine, error) {
	if ok, retryAfter := s.attempts.Allow("join:" + userID.String()); !ok {
		return Mine{}, apperr.RateLimited(retryAfter)
	}
	code = normalizeInviteCode(code)
	if code == "" {
		return Mine{}, apperr.Validation(map[string]string{
			"code": "Enter an invite code.",
		})
	}
	if err := s.repo.Join(ctx, userID, code, s.now()); err != nil {
		return Mine{}, err
	}
	s.events.CouplePaired()
	return s.GetMine(ctx, userID)
}

func normalizeInviteCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Service) GetMine(ctx context.Context, userID uuid.UUID) (Mine, error) {
	return s.repo.GetForUser(ctx, userID, s.now())
}

func (s *Service) UpdateCouples(ctx context.Context, userID uuid.UUID, update UpdateDto) (Mine, error) {
	start, err := parseDay(update.RelationshipStartDate)
	if err != nil {
		return Mine{}, err
	}
	// The zone moves for both partners at once (DEC-27); validated before anything is written.
	var timezone *string
	if update.Timezone != nil {
		valid, err := user.ValidateTimezone(*update.Timezone)
		if err != nil {
			return Mine{}, err
		}
		timezone = &valid
	}
	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if err := s.repo.UpdateCouples(ctx, mine.Couple.ID, start, update.Name, timezone); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdateRole(ctx context.Context, userID uuid.UUID, role string) (Mine, error) {
	role, err := ValidateRole(role)
	if err != nil {
		return Mine{}, err
	}

	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if err := s.repo.UpdateRole(ctx, userID, mine.Couple.ID, role); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

// UpdateOnboarding leaves a nil field alone so the client can send one step at a time.
func (s *Service) UpdateOnboarding(ctx context.Context, userID uuid.UUID, patch OnboardingDto) (Mine, error) {
	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if patch.Install == nil && patch.Notifications == nil {
		return mine, nil
	}
	if err := s.repo.UpdateOnboarding(ctx, userID, mine.Couple.ID, patch.Install, patch.Notifications); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

// LeaveCouple reads the couple id from the caller's own membership, like
// every other write here, so a client can't substitute a different id.
func (s *Service) LeaveCouple(ctx context.Context, userID uuid.UUID) ([]Mine, error) {
	if err := s.repo.Dissolve(ctx, userID, s.now()); err != nil {
		return nil, err
	}
	s.events.CoupleEnded()
	return s.Archived(ctx, userID)
}

func (s *Service) Archived(ctx context.Context, userID uuid.UUID) ([]Mine, error) {
	return s.repo.GetArchivedForUser(ctx, userID, s.now())
}
